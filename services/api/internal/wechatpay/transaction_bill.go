package wechatpay

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const billResponseMaxBytes = 8 * 1024 * 1024

type BillEntryKind string

const (
	BillEntryPayment BillEntryKind = "PAYMENT"
	BillEntryRefund  BillEntryKind = "REFUND"
)

// BillEntry is one provider transaction-bill fact before domain filtering.
type BillEntry struct {
	Kind        BillEntryKind
	OutTradeNo  string
	OutRefundNo string
	ProviderID  string
	AmountCents uint64
	Currency    string
	State       string
	OccurredAt  time.Time
}

// TransactionBill is one verified-integrity provider file and its strict entries.
type TransactionBill struct {
	Date    time.Time
	Digest  [32]byte
	Entries []BillEntry
}

type transactionBillApplyResponse struct {
	HashType    string `json:"hash_type"`
	HashValue   string `json:"hash_value"`
	DownloadURL string `json:"download_url"`
}

var allTransactionBillHeader = []string{
	"交易时间", "公众账号ID", "商户号", "特约商户号", "设备号", "微信订单号", "商户订单号", "用户标识",
	"交易类型", "交易状态", "付款银行", "货币种类", "应结订单金额", "代金券金额", "微信退款单号", "商户退款单号",
	"退款金额", "充值券退款金额", "退款类型", "退款状态", "商品名称", "商户数据包", "手续费", "费率", "订单金额", "申请退款金额", "费率备注",
}

var allTransactionBillSummaryHeader = []string{
	"总交易单数", "应结订单总金额", "退款总金额", "充值券退款总金额", "手续费总金额", "订单总金额", "申请退款总金额",
}

// DownloadTransactionBill requests an uncompressed ALL bill, verifies its official SHA1, and strictly parses it.
func (client *Client) DownloadTransactionBill(ctx context.Context, date time.Time) (TransactionBill, error) {
	if client == nil || date.IsZero() {
		return TransactionBill{}, &Error{kind: ErrorProtocol}
	}
	date = normalizedTransactionBillDate(date)
	query := url.Values{"bill_date": []string{date.Format("2006-01-02")}, "bill_type": []string{"ALL"}}
	requestTarget := "/v3/bill/tradebill?" + query.Encode()
	body, err := client.sendSignedRequest(ctx, http.MethodGet, requestTarget, nil, http.StatusOK)
	if err != nil {
		return TransactionBill{}, err
	}
	var apply transactionBillApplyResponse
	if err := decodeStrictJSON(body, &apply); err != nil || apply.HashType != "SHA1" || apply.HashValue == "" || apply.DownloadURL == "" {
		return TransactionBill{}, &Error{kind: ErrorProtocol}
	}
	expectedHash, err := hex.DecodeString(apply.HashValue)
	if err != nil || len(expectedHash) != sha1.Size {
		return TransactionBill{}, &Error{kind: ErrorProtocol}
	}
	downloadURL, requestTarget, err := validatedBillDownloadURL(apply.DownloadURL)
	if err != nil {
		return TransactionBill{}, err
	}
	billBody, err := client.downloadBill(ctx, downloadURL, requestTarget)
	if err != nil {
		return TransactionBill{}, err
	}
	actualHash := sha1.Sum(billBody)
	if subtle.ConstantTimeCompare(expectedHash, actualHash[:]) != 1 {
		return TransactionBill{}, &Error{kind: ErrorProtocol}
	}
	entries, err := parseTransactionBill(billBody, client.appID, client.merchantID)
	if err != nil {
		return TransactionBill{}, err
	}
	return TransactionBill{Date: date, Digest: sha256.Sum256(billBody), Entries: entries}, nil
}

func validatedBillDownloadURL(raw string) (*url.URL, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "api.mch.weixin.qq.com" || parsed.User != nil ||
		parsed.Fragment != "" || parsed.Opaque != "" || parsed.RawQuery == "" {
		return nil, "", &Error{kind: ErrorProtocol}
	}
	switch parsed.EscapedPath() {
	case "/v3/billdownload/file", "/v3/bill/downloadurl":
	default:
		return nil, "", &Error{kind: ErrorProtocol}
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 1 || len(query["token"]) != 1 || query.Get("token") == "" {
		return nil, "", &Error{kind: ErrorProtocol}
	}
	return parsed, parsed.EscapedPath() + "?" + parsed.RawQuery, nil
}

func (client *Client) downloadBill(ctx context.Context, target *url.URL, requestTarget string) ([]byte, error) {
	timestamp := strconv.FormatInt(client.now().Unix(), 10)
	nonce, err := client.nonce()
	if err != nil || !safeHeaderToken(nonce) {
		return nil, &Error{kind: ErrorProtocol}
	}
	message := http.MethodGet + "\n" + requestTarget + "\n" + timestamp + "\n" + nonce + "\n\n"
	signature, err := signSHA256RSA(client.merchantPrivateKey, []byte(message))
	if err != nil {
		return nil, err
	}
	authorization := fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		client.merchantID, nonce, signature, timestamp, client.merchantSerial,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, &Error{kind: ErrorProtocol}
	}
	request.Header.Set("Accept", "text/plain")
	request.Header.Set("Authorization", authorization)
	response, err := client.httpClient.Do(request)
	if err != nil {
		var networkError net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
			return nil, &Error{kind: ErrorTimeout}
		}
		return nil, &Error{kind: ErrorTransport}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		kind := ErrorProviderRejected
		if response.StatusCode == http.StatusTooManyRequests {
			kind = ErrorRateLimited
		} else if response.StatusCode >= http.StatusInternalServerError {
			kind = ErrorProviderUnavailable
		} else if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
			kind = ErrorProtocol
		}
		return nil, &Error{kind: kind, statusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, billResponseMaxBytes+1))
	if err != nil || len(body) > billResponseMaxBytes {
		return nil, &Error{kind: ErrorProtocol}
	}
	return body, nil
}

func parseTransactionBill(body []byte, appID, merchantID string) ([]BillEntry, error) {
	if len(body) == 0 || appID == "" || merchantID == "" {
		return nil, &Error{kind: ErrorProtocol}
	}
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil || !sameBillFields(header, allTransactionBillHeader) {
		return nil, &Error{kind: ErrorProtocol}
	}

	entries := make([]BillEntry, 0)
	detailCount := uint64(0)
	for {
		record, err := reader.Read()
		if err != nil {
			return nil, &Error{kind: ErrorProtocol}
		}
		if len(record) == 1 && record[0] == "" {
			continue
		}
		if sameBillFields(record, allTransactionBillSummaryHeader) {
			break
		}
		if len(record) != len(allTransactionBillHeader) || !stripBillMarkers(record) {
			return nil, &Error{kind: ErrorProtocol}
		}
		entry, err := parseTransactionBillEntry(record, appID, merchantID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		detailCount++
	}

	summary, err := reader.Read()
	if err != nil || len(summary) != len(allTransactionBillSummaryHeader) || !stripBillMarkers(summary) {
		return nil, &Error{kind: ErrorProtocol}
	}
	summaryCount, err := strconv.ParseUint(summary[0], 10, 64)
	if err != nil || summaryCount != detailCount {
		return nil, &Error{kind: ErrorProtocol}
	}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || !(len(record) == 1 && record[0] == "") {
			return nil, &Error{kind: ErrorProtocol}
		}
	}
	return entries, nil
}

func parseTransactionBillEntry(record []string, appID, merchantID string) (BillEntry, error) {
	if record[1] != appID || record[2] != merchantID || record[5] == "" || record[6] == "" || record[11] != "CNY" {
		return BillEntry{}, &Error{kind: ErrorProtocol}
	}
	location := time.FixedZone("CST", 8*60*60)
	occurredAt, err := time.ParseInLocation("2006-01-02 15:04:05", record[0], location)
	if err != nil {
		return BillEntry{}, &Error{kind: ErrorProtocol}
	}
	entry := BillEntry{OutTradeNo: record[6], Currency: record[11], State: record[9], OccurredAt: occurredAt.UTC()}
	switch record[9] {
	case "SUCCESS":
		entry.Kind = BillEntryPayment
		entry.ProviderID = record[5]
		entry.AmountCents, err = parseBillYuan(record[24])
	case "REFUND":
		entry.Kind = BillEntryRefund
		entry.ProviderID = record[14]
		entry.OutRefundNo = record[15]
		entry.State = record[19]
		if entry.ProviderID == "" || entry.OutRefundNo == "" || !validBillRefundState(entry.State) {
			return BillEntry{}, &Error{kind: ErrorProtocol}
		}
		entry.AmountCents, err = parseBillYuan(record[25])
	default:
		return BillEntry{}, &Error{kind: ErrorProtocol}
	}
	if err != nil || entry.AmountCents == 0 {
		return BillEntry{}, &Error{kind: ErrorProtocol}
	}
	return entry, nil
}

func validBillRefundState(state string) bool {
	switch state {
	case "SUCCESS", "PROCESSING", "FAIL", "CHANGE":
		return true
	default:
		return false
	}
}

func parseBillYuan(value string) (uint64, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 || len(parts[1]) != 2 || parts[0] == "" || len(parts[0]) > 18 {
		return 0, &Error{kind: ErrorProtocol}
	}
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, &Error{kind: ErrorProtocol}
	}
	fraction, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil || fraction > 99 || whole > (math.MaxUint64-fraction)/100 {
		return 0, &Error{kind: ErrorProtocol}
	}
	return whole*100 + fraction, nil
}

func sameBillFields(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		value := strings.TrimPrefix(got[index], "\ufeff")
		value = strings.TrimPrefix(value, "`")
		if value != want[index] {
			return false
		}
	}
	return true
}

func stripBillMarkers(record []string) bool {
	for index := range record {
		if !strings.HasPrefix(record[index], "`") {
			return false
		}
		record[index] = strings.TrimPrefix(record[index], "`")
	}
	return true
}

func normalizedTransactionBillDate(date time.Time) time.Time {
	date = date.UTC()
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
