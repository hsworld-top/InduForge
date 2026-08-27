package service

import (
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNormalizeSQLValuePreservesPostgresTypes(t *testing.T) {
	numeric := pgtype.Numeric{Int: big.NewInt(999999999999991234), Exp: -4, Valid: true}
	if actual := normalizeSQLValue(numeric, "numeric"); actual != "99999999999999.1234" {
		t.Fatalf("unexpected numeric value: %#v", actual)
	}

	uuidValue := pgtype.UUID{Bytes: [16]byte{0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00}, Valid: true}
	if actual := normalizeSQLValue(uuidValue, "uuid"); actual != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected uuid value: %#v", actual)
	}
	if actual := normalizeSQLValue(uuidValue.Bytes, "uuid"); actual != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected uuid byte array: %#v", actual)
	}
	if actual := normalizeSQLValue(uuidValue.Bytes[:], "UNIQUEIDENTIFIER"); actual != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected SQL Server uniqueidentifier: %#v", actual)
	}

	timeValue := pgtype.Time{Microseconds: ((12*60+34)*60 + 56) * 1_000_000, Valid: true}
	if actual := normalizeSQLValue(timeValue, "time"); actual != "12:34:56" {
		t.Fatalf("unexpected time value: %#v", actual)
	}

	if actual := normalizeSQLValue([]byte{0x00, 0xff, 0x10}, "bytea"); actual != "AP8Q" {
		t.Fatalf("unexpected bytea value: %#v", actual)
	}
}
