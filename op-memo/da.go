package memo

// DerivationVersionMemo is a byte marker for memo references submitted
// to the batch inbox address as calldata.
// Mnemonic 0xda = memo
//
// version 0xda references are encoded as:
// [1]byte marker + [32]byte mid
// in little-endian encoding.
const DerivationVersionMemo = 0xda
