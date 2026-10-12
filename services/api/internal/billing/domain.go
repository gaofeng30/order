package billing

import "sort"

func CompareBill(provider, system []BillEntry) ReconcileResult {
	providerByKey := make(map[string]BillEntry, len(provider))
	systemByKey := make(map[string]BillEntry, len(system))
	for _, entry := range provider {
		providerByKey[entryKey(entry)] = entry
	}
	for _, entry := range system {
		systemByKey[entryKey(entry)] = entry
	}
	result := ReconcileResult{}
	for key, entry := range providerByKey {
		systemEntry, ok := systemByKey[key]
		if ok && sameEntry(entry, systemEntry) {
			result.Matched++
			delete(systemByKey, key)
			continue
		}
		result.ProviderOnly = append(result.ProviderOnly, entry)
	}
	for _, entry := range systemByKey {
		result.SystemOnly = append(result.SystemOnly, entry)
	}
	sortEntries(result.ProviderOnly)
	sortEntries(result.SystemOnly)
	return result
}

func entryKey(entry BillEntry) string {
	if entry.Kind == EntryRefund {
		return string(entry.Kind) + "\x00" + entry.OutRefundNo
	}
	return string(entry.Kind) + "\x00" + entry.OutTradeNo
}

func sameEntry(left, right BillEntry) bool {
	return left.Kind == right.Kind && left.OutTradeNo == right.OutTradeNo && left.OutRefundNo == right.OutRefundNo &&
		left.ProviderID == right.ProviderID && left.AmountCents == right.AmountCents && left.Currency == right.Currency && left.State == right.State
}

func sortEntries(entries []BillEntry) {
	sort.Slice(entries, func(i, j int) bool { return entryKey(entries[i]) < entryKey(entries[j]) })
}
