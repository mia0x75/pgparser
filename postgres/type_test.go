package postgres

import (
	"testing"
)

func TestFlags(t *testing.T) {
	if !HasNotNullFlag(NotNullFlag) {
		t.Fatal("HasNotNullFlag(NotNullFlag) is false")
	}
	if !HasUniKeyFlag(UniqueKeyFlag) {
		t.Fatal("HasUniKeyFlag(UniqueKeyFlag) is false")
	}
	if !HasNotNullFlag(NotNullFlag) {
		t.Fatal("HasNotNullFlag(NotNullFlag) is false")
	}
	if !HasNoDefaultValueFlag(NoDefaultValueFlag) {
		t.Fatal("HasNoDefaultValueFlag(NoDefaultValueFlag) is false")
	}
	if !HasAutoIncrementFlag(AutoIncrementFlag) {
		t.Fatal("HasAutoIncrementFlag(AutoIncrementFlag) is false")
	}
	if !HasUnsignedFlag(UnsignedFlag) {
		t.Fatal("HasUnsignedFlag(UnsignedFlag) is false")
	}
	if !HasZerofillFlag(ZerofillFlag) {
		t.Fatal("HasZerofillFlag(ZerofillFlag) is false")
	}
	if !HasBinaryFlag(BinaryFlag) {
		t.Fatal("HasBinaryFlag(BinaryFlag) is false")
	}
	if !HasPriKeyFlag(PriKeyFlag) {
		t.Fatal("HasPriKeyFlag(PriKeyFlag) is false")
	}
	if !HasMultipleKeyFlag(MultipleKeyFlag) {
		t.Fatal("HasMultipleKeyFlag(MultipleKeyFlag) is false")
	}
	if !HasTimestampFlag(TimestampFlag) {
		t.Fatal("HasTimestampFlag(TimestampFlag) is false")
	}
	if !HasOnUpdateNowFlag(OnUpdateNowFlag) {
		t.Fatal("HasOnUpdateNowFlag(OnUpdateNowFlag) is false")
	}
}
