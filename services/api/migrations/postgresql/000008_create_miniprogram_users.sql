CREATE SEQUENCE miniprogram_users_id_seq AS BIGINT;
CREATE TABLE miniprogram_users (
    id NUMERIC(20,0) DEFAULT nextval('miniprogram_users_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
    openid BYTEA CHECK (octet_length(openid) <= 128) NOT NULL,
    created_at TIMESTAMPTZ(6) NOT NULL,
    last_login_at TIMESTAMPTZ(6) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT uq_miniprogram_users_openid UNIQUE (openid)
);
ALTER SEQUENCE miniprogram_users_id_seq OWNED BY miniprogram_users.id;
