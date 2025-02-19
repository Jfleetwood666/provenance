package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"sigs.k8s.io/yaml"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

const (
	// PrefixScope is the address human readable prefix used with bech32 encoding of Scope IDs
	PrefixScope = "scope"
	// PrefixSession is the address human readable prefix used with bech32 encoding of Session IDs
	PrefixSession = "session"
	// PrefixRecord is the address human readable prefix used with bech32 encoding of Record IDs
	PrefixRecord = "record"

	// PrefixScopeSpecification is the address human readable prefix used with bech32 encoding of ScopeSpecification IDs
	PrefixScopeSpecification = "scopespec"
	// PrefixContractSpecification is the address human readable prefix used with bech32 encoding of ContractSpecification IDs
	PrefixContractSpecification = "contractspec"
	// PrefixRecordSpecification is the address human readable prefix used with bech32 encoding of RecordSpecification IDs
	PrefixRecordSpecification = "recspec"

	// DenomPrefix is the string prepended to a metadata address to create the denom for that metadata object.
	DenomPrefix = "nft/"
)

var (
	// Ensure MetadataAddress implements the sdk.Address interface
	_ sdk.Address = MetadataAddress{}
)

// MetadataAddress is a blockchain compliant address based on UUIDs
type MetadataAddress []byte

// VerifyMetadataAddressFormat checks a sequence of bytes for proper format as a MetadataAddress instance
// returns the associated bech32 hrp/type name or any errors encountered during verification
func VerifyMetadataAddressFormat(bz []byte) (string, error) {
	hrp := ""
	if len(bz) == 0 {
		return hrp, errors.New("address is empty")
	}
	var requiredLength int
	checkSecondaryUUID := false
	switch bz[0] {
	case ScopeKeyPrefix[0]:
		hrp = PrefixScope
		requiredLength = 1 + 16 // type byte plus size of one uuid
	case SessionKeyPrefix[0]:
		hrp = PrefixSession
		requiredLength = 1 + 16 + 16 // type byte plus size of two uuids
		checkSecondaryUUID = true
	case RecordKeyPrefix[0]:
		hrp = PrefixRecord
		requiredLength = 1 + 16 + 16 // type byte plus size of one uuid and one half sha256 hash

	case ScopeSpecificationKeyPrefix[0]:
		hrp = PrefixScopeSpecification
		requiredLength = 1 + 16 // type byte plus size of one uuid
	case ContractSpecificationKeyPrefix[0]:
		hrp = PrefixContractSpecification
		requiredLength = 1 + 16 // type byte plus size of one uuid
	case RecordSpecificationKeyPrefix[0]:
		hrp = PrefixRecordSpecification
		requiredLength = 1 + 16 + 16 // type byte plus size of one uuid plus one-half sha256 hash

	default:
		return hrp, fmt.Errorf("invalid metadata address type: %d", bz[0])
	}
	if len(bz) != requiredLength {
		return hrp, fmt.Errorf("incorrect address length (expected: %d, actual: %d)", requiredLength, len(bz))
	}
	// all valid metadata address have at least one uuid
	if _, err := uuid.FromBytes(bz[1:17]); err != nil {
		return hrp, fmt.Errorf("invalid address bytes of uuid, expected uuid compliant: %w", err)
	}
	if checkSecondaryUUID {
		if _, err := uuid.FromBytes(bz[17:33]); err != nil {
			return hrp, fmt.Errorf("invalid address bytes of secondary uuid, expected uuid compliant: %w", err)
		}
	}
	return hrp, nil
}

// getNameForHRP returns the more formal name used for each metadata hrp.
// E.g. if the hrp is PrefixRecordSpecification (i.e. "recspec"), this will return "record specification".
func getNameForHRP(hrp string) string {
	switch hrp {
	case PrefixScope, PrefixSession, PrefixRecord:
		return hrp
	case PrefixScopeSpecification, PrefixContractSpecification:
		return strings.TrimSuffix(hrp, "spec") + " specification"
	case PrefixRecordSpecification:
		return "record specification"
	}
	return fmt.Sprintf("<%q>", hrp)
}

// VerifyMetadataAddressHasType makes sure that the provided ma is a valid MetadataAddress
// and that it has the provided type (e.g. PrefixScope or PrefixContractSpecification, etc.).
func VerifyMetadataAddressHasType(ma MetadataAddress, expHRP string) error {
	hrp, err := VerifyMetadataAddressFormat(ma)
	if err != nil {
		return fmt.Errorf("invalid %s metadata address %#v: %w", getNameForHRP(expHRP), ma, err)
	}
	if hrp != expHRP {
		return fmt.Errorf("invalid %s id %q: wrong type", getNameForHRP(expHRP), ma)
	}
	return nil
}

// ConvertHashToAddress constructs a MetadataAddress using the provided type code and the raw bytes of the
// base64 decoded hash, limited appropriately by the desired typeCode.
// The resulting Address is not guaranteed to contain valid UUIDS or name hashes.
func ConvertHashToAddress(typeCode []byte, hash string) (MetadataAddress, error) {
	var addr MetadataAddress
	var err error
	if len(typeCode) == 0 {
		return addr, errors.New("empty typeCode bytes")
	}
	if len(hash) == 0 {
		return addr, errors.New("empty hash string")
	}
	reqLen := 0
	switch typeCode[0] {
	case ScopeKeyPrefix[0], ContractSpecificationKeyPrefix[0], ScopeSpecificationKeyPrefix[0]:
		// Scopes, ContractSpecs, and ScopeSpecs are a type byte followed by 16 bytes (usually a uuid)
		reqLen = 16
	case SessionKeyPrefix[0]:
		// Sessions are a type byte followed by 32 bytes (usually two uuids)
		reqLen = 32
	case RecordKeyPrefix[0], RecordSpecificationKeyPrefix[0]:
		// Records and Record specifications are a type byte followed by 16 bytes (usually a uuid) followed by 16 bytes (from the hashed name value)
		reqLen = 32
	default:
		return addr, fmt.Errorf("invalid address type code 0x%X", typeCode)
	}
	var raw []byte
	raw, err = base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return addr, err
	}
	if len(raw) < reqLen {
		return addr, fmt.Errorf("invalid hash \"%s\" byte length, expected at least %d bytes, found %d",
			hash, reqLen, len(raw))
	}
	err = addr.Unmarshal(append([]byte{typeCode[0]}, raw[0:reqLen]...))
	return addr, err
}

// MetadataAddressFromHex creates a MetadataAddress from a hex string.  NOTE: Does not perform validation on address,
// only performs basic HEX decoding checks.  This method matches the sdk.AccAddress approach
func MetadataAddressFromHex(address string) (MetadataAddress, error) {
	if len(address) == 0 {
		return MetadataAddress{}, errors.New("address decode failed: must provide an address")
	}
	bz, err := hex.DecodeString(address)
	return MetadataAddress(bz), err
}

// ParseMetadataAddressFromBech32 creates a MetadataAddress from a Bech32 string.
// The encoded data is checked against the provided bech32 hrp along with an overall verification of the byte format.
// The address, hrp, and any error are returned.
//
// If you don't need the HRP, use MetadataAddressFromBech32.
func ParseMetadataAddressFromBech32(address string) (MetadataAddress, string, error) {
	if len(strings.TrimSpace(address)) == 0 {
		return MetadataAddress{}, "", errors.New("empty address string is not allowed")
	}

	hrp, bz, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return nil, "", err
	}
	expectedHrp, err := VerifyMetadataAddressFormat(bz)
	if err != nil {
		return nil, "", err
	}
	if expectedHrp != hrp {
		return MetadataAddress{}, "", fmt.Errorf("invalid bech32 prefix; expected %s, got %s", expectedHrp, hrp)
	}

	return bz, hrp, nil
}

// MetadataAddressFromBech32 creates a MetadataAddress from a Bech32 string.  The encoded data is checked against the
// provided bech32 hrp along with an overall verification of the byte format.
//
// If you need the HRP, use ParseMetadataAddressFromBech32.
func MetadataAddressFromBech32(address string) (addr MetadataAddress, err error) {
	bz, _, err := ParseMetadataAddressFromBech32(address)
	return bz, err
}

// MetadataAddressFromDenom gets the MetadataAddress that the provided denom is for.
func MetadataAddressFromDenom(denom string) (MetadataAddress, error) {
	id := strings.TrimPrefix(denom, DenomPrefix)
	if id == denom {
		return nil, fmt.Errorf("denom %q is not a MetadataAddress denom", denom)
	}
	rv, err := MetadataAddressFromBech32(id)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata address in denom %q: %w", denom, err)
	}
	return rv, nil
}

// ScopeMetadataAddress creates a MetadataAddress instance for the given scope by its uuid
func ScopeMetadataAddress(scopeUUID uuid.UUID) MetadataAddress {
	bz, err := scopeUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return append(ScopeKeyPrefix, bz...)
}

// SessionMetadataAddress creates a MetadataAddress instance for a session within a scope by uuids
func SessionMetadataAddress(scopeUUID uuid.UUID, sessionUUID uuid.UUID) MetadataAddress {
	bz, err := scopeUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	addr := SessionKeyPrefix
	addr = append(addr, bz...)
	bz, err = sessionUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return append(addr, bz...)
}

// RecordMetadataAddress creates a MetadataAddress instance for a record within a scope by scope uuid/record name
func RecordMetadataAddress(scopeUUID uuid.UUID, name string) MetadataAddress {
	bz, err := scopeUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	addr := RecordKeyPrefix
	addr = append(addr, bz...)
	name = strings.ToLower(strings.TrimSpace(name))
	if len(name) < 1 {
		panic("missing name value for record metadata address")
	}
	nameBytes := sha256.Sum256([]byte(name))
	return append(addr, nameBytes[0:16]...)
}

// ScopeSpecMetadataAddress creates a MetadataAddress instance for a scope specification
func ScopeSpecMetadataAddress(specUUID uuid.UUID) MetadataAddress {
	bz, err := specUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return append(ScopeSpecificationKeyPrefix, bz...)
}

// ContractSpecMetadataAddress creates a MetadataAddress instance for a contract specification
func ContractSpecMetadataAddress(specUUID uuid.UUID) MetadataAddress {
	bz, err := specUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return append(ContractSpecificationKeyPrefix, bz...)
}

// RecordSpecMetadataAddress creates a MetadataAddress instance for a record specification
func RecordSpecMetadataAddress(contractSpecUUID uuid.UUID, name string) MetadataAddress {
	bz, err := contractSpecUUID.MarshalBinary()
	if err != nil {
		panic(err)
	}
	addr := RecordSpecificationKeyPrefix
	addr = append(addr, bz...)
	name = strings.ToLower(strings.TrimSpace(name))
	if len(name) < 1 {
		panic("missing name value for record spec metadata address")
	}
	nameBytes := sha256.Sum256([]byte(name))
	return append(addr, nameBytes[0:16]...)
}

// Equals determines if the current MetadataAddress is equal to another sdk.Address
func (ma MetadataAddress) Equals(ma2 sdk.Address) bool {
	if ma.Empty() && ma2.Empty() {
		return true
	}
	return bytes.Equal(ma.Bytes(), ma2.Bytes())
}

// Empty returns true if the MetadataAddress is uninitialized
func (ma MetadataAddress) Empty() bool {
	if ma == nil {
		return true
	}

	ma2 := MetadataAddress{}
	return bytes.Equal(ma.Bytes(), ma2.Bytes())
}

// Validate determines if the contained bytes form a valid MetadataAddress according to its type
func (ma MetadataAddress) Validate() (err error) {
	_, err = VerifyMetadataAddressFormat(ma)
	return
}

// Marshal returns the bytes underlying the MetadataAddress instance
func (ma MetadataAddress) Marshal() ([]byte, error) {
	return ma, nil
}

// Unmarshal initializes a MetadataAddress instance using the given bytes.  An error will be returned if the
// given bytes do not form a valid Address
func (ma *MetadataAddress) Unmarshal(data []byte) error {
	*ma = data
	if len(data) == 0 {
		return nil
	}
	_, err := VerifyMetadataAddressFormat(data)
	return err
}

// MarshalJSON returns a JSON representation for the current address using a bech32 encoded string
func (ma MetadataAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(ma.String())
}

// MarshalYAML returns a YAML representation for the current address using a bech32 encoded string
func (ma MetadataAddress) MarshalYAML() (interface{}, error) {
	return ma.String(), nil
}

// UnmarshalJSON creates a MetadataAddress instance from the given JSON data
func (ma *MetadataAddress) UnmarshalJSON(data []byte) error {
	var s string

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*ma = MetadataAddress{}
		return nil
	}

	ma2, err := MetadataAddressFromBech32(s)
	if err != nil {
		return err
	}

	*ma = ma2
	return nil
}

// UnmarshalYAML creates a MetadataAddress instance from the given YAML data
func (ma *MetadataAddress) UnmarshalYAML(data []byte) error {
	var s string
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*ma = MetadataAddress{}
		return nil
	}

	ma2, err := MetadataAddressFromBech32(s)
	if err != nil {
		return err
	}

	*ma = ma2
	return nil
}

// Bytes implements Address interface, returns the raw bytes for this Address
func (ma MetadataAddress) Bytes() []byte {
	return ma
}

// String implements the stringer interface and encodes as a bech32
func (ma MetadataAddress) String() string {
	if ma.Empty() {
		return ""
	}

	hrp, err := VerifyMetadataAddressFormat(ma)
	if err != nil {
		// Be careful changing this. The %#v path in MetadataAddress.Format does NOT call this String method,
		// but %v, %q and %s do. So there would be infinite recursion with some other formatter directives.
		return fmt.Sprintf("%#v", ma)
	}

	bech32Addr, err := bech32.ConvertAndEncode(hrp, ma.Bytes())
	if err != nil {
		panic(err)
	}

	return bech32Addr
}

// Size implements gogoproto custom type interface and returns the number of bytes in this instance
func (ma MetadataAddress) Size() int {
	return len(ma)
}

// MarshalTo implements gogoproto custom type interface and writes the current bytes into the provided data structure
func (ma *MetadataAddress) MarshalTo(data []byte) (int, error) {
	if len(*ma) == 0 {
		return 0, nil
	}
	copy(data, *ma)
	return len(*ma), nil
}

// Compare exists to fit gogoprotobuf custom type interface.
func (ma MetadataAddress) Compare(other MetadataAddress) int {
	return bytes.Compare(ma[0:], other[0:])
}

// ScopeUUID returns the scope uuid component of a MetadataAddress (if appropriate)
func (ma MetadataAddress) ScopeUUID() (uuid.UUID, error) {
	if !ma.isTypeOneOf(ScopeKeyPrefix, SessionKeyPrefix, RecordKeyPrefix) {
		return uuid.UUID{}, fmt.Errorf("this metadata address (%s) does not contain a scope uuid", ma)
	}
	return ma.PrimaryUUID()
}

// SessionUUID returns the session uuid component of a MetadataAddress (if appropriate)
func (ma MetadataAddress) SessionUUID() (uuid.UUID, error) {
	if len(ma) > 0 && ma[0] != SessionKeyPrefix[0] {
		return uuid.UUID{}, fmt.Errorf("this metadata address (%s) does not contain a session uuid", ma)
	}
	return ma.SecondaryUUID()
}

// ScopeSpecUUID returns the scope specification uuid component of a MetadataAddress (if appropriate)
func (ma MetadataAddress) ScopeSpecUUID() (uuid.UUID, error) {
	if len(ma) > 0 && ma[0] != ScopeSpecificationKeyPrefix[0] {
		return uuid.UUID{}, fmt.Errorf("this metadata address (%s) does not contain a scope specification uuid", ma)
	}
	return ma.PrimaryUUID()
}

// ContractSpecUUID returns the contract specification uuid component of a MetadataAddress (if appropriate)
func (ma MetadataAddress) ContractSpecUUID() (uuid.UUID, error) {
	if !ma.isTypeOneOf(ContractSpecificationKeyPrefix, RecordSpecificationKeyPrefix) {
		return uuid.UUID{}, fmt.Errorf("this metadata address (%s) does not contain a contract specification uuid", ma)
	}
	return ma.PrimaryUUID()
}

// Prefix returns the human readable part (prefix) of this MetadataAddress, e.g. "scope" or "contractspec"
// More accurately, this converts the 1st byte into its human readable string value.
func (ma MetadataAddress) Prefix() (string, error) {
	prefix, err := VerifyMetadataAddressFormat(ma)
	if len(prefix) == 0 {
		return prefix, err
	}
	return prefix, nil
}

// PrimaryUUID returns the primary UUID from this MetadataAddress (if applicable).
// More accurately, this converts bytes 2 to 17 to a UUID.
// For example, if this MetadataAddress is for a scope specification, this will return the scope specification uuid.
// But if this MetadataAddress is for a record specification, this will return the contract specification
// (since that's the first part of those metadata addresses).
func (ma MetadataAddress) PrimaryUUID() (uuid.UUID, error) {
	if len(ma) < 1 {
		return uuid.UUID{}, fmt.Errorf("address empty")
	}
	// if we don't know this type
	if !ma.isTypeOneOf(ScopeKeyPrefix, SessionKeyPrefix, RecordKeyPrefix, ScopeSpecificationKeyPrefix, ContractSpecificationKeyPrefix, RecordSpecificationKeyPrefix) {
		return uuid.UUID{}, fmt.Errorf("invalid address type out of valid range (got: %d)", ma[0])
	}
	if len(ma) < 17 {
		return uuid.UUID{}, fmt.Errorf("incorrect address length (must be at least 17, actual: %d)", len(ma))
	}
	return uuid.FromBytes(ma[1:17])
}

// SecondaryUUID returns the secondary UUID from this MetadataAddress (if applicable).
// More accurately, this converts bytes 18 to 33 (inclusive) to a UUID.
func (ma MetadataAddress) SecondaryUUID() (uuid.UUID, error) {
	if len(ma) < 1 {
		return uuid.UUID{}, fmt.Errorf("address empty")
	}
	// if we don't know this type
	if !ma.isTypeOneOf(SessionKeyPrefix) {
		return uuid.UUID{}, fmt.Errorf("invalid address type out of valid range (got: %d)", ma[0])
	}
	if len(ma) < 33 {
		return uuid.UUID{}, fmt.Errorf("incorrect address length (must be at least 33, actual: %d)", len(ma))
	}
	return uuid.FromBytes(ma[17:33])
}

// NameHash returns the hashed name bytes from this MetadataAddress (if applicable).
// More accurately, this returns a copy of bytes 18 through 33 (inclusive).
func (ma MetadataAddress) NameHash() ([]byte, error) {
	namehash := make([]byte, 16)
	if len(ma) < 1 {
		return namehash, fmt.Errorf("address empty")
	}
	if !ma.isTypeOneOf(RecordKeyPrefix, RecordSpecificationKeyPrefix) {
		return namehash, fmt.Errorf("invalid address type out of valid range (got: %d)", ma[0])
	}
	if len(ma) < 33 {
		return namehash, fmt.Errorf("incorrect address length (must be at least 33, actual: %d)", len(ma))
	}
	copy(namehash, ma[17:])
	return namehash, nil
}

// AsScopeAddress returns the MetadataAddress for a scope using the scope UUID within the current context
func (ma MetadataAddress) AsScopeAddress() (MetadataAddress, error) {
	scopeUUID, err := ma.ScopeUUID()
	if err != nil {
		return MetadataAddress{}, err
	}
	return ScopeMetadataAddress(scopeUUID), nil
}

// MustGetAsScopeAddress returns the MetadataAddress for a scope using the scope UUID within the current context
// This is the same as AsScopeAddress except it panics on error.
func (ma MetadataAddress) MustGetAsScopeAddress() MetadataAddress {
	retval, err := ma.AsScopeAddress()
	if err != nil {
		panic(err)
	}
	return retval
}

// AsSessionAddress returns the MetadataAddress for a session using the scope UUID within the current context and the provided session UUID.
func (ma MetadataAddress) AsSessionAddress(sessionUUID uuid.UUID) (MetadataAddress, error) {
	scopeUUID, err := ma.ScopeUUID()
	if err != nil {
		return MetadataAddress{}, err
	}
	return SessionMetadataAddress(scopeUUID, sessionUUID), nil
}

// MustGetAsSessionAddress returns the MetadataAddress for a session using the scope UUID within the current context and the provided session UUID.
// This is the same as AsSessionAddress except it panics on error.
func (ma MetadataAddress) MustGetAsSessionAddress(sessionUUID uuid.UUID) MetadataAddress {
	retval, err := ma.AsSessionAddress(sessionUUID)
	if err != nil {
		panic(err)
	}
	return retval
}

// AsRecordAddress returns the MetadataAddress for a record using the scope UUID within the current context and the provided name
func (ma MetadataAddress) AsRecordAddress(name string) (MetadataAddress, error) {
	scopeUUID, err := ma.ScopeUUID()
	if err != nil {
		return MetadataAddress{}, err
	}
	if len(name) == 0 {
		return MetadataAddress{}, errors.New("missing name value for record metadata address")
	}
	return RecordMetadataAddress(scopeUUID, name), nil
}

// MustGetAsRecordAddress returns the MetadataAddress for a record using the scope UUID within the current context and the provided name
// This is the same as AsRecordAddress except it panics on error.
func (ma MetadataAddress) MustGetAsRecordAddress(name string) MetadataAddress {
	retval, err := ma.AsRecordAddress(name)
	if err != nil {
		panic(err)
	}
	return retval
}

// AsRecordSpecAddress returns the MetadataAddress for a record spec using the contract spec UUID within the current context and the provided name
func (ma MetadataAddress) AsRecordSpecAddress(name string) (MetadataAddress, error) {
	contractSpecUUID, err := ma.ContractSpecUUID()
	if err != nil {
		return MetadataAddress{}, err
	}
	return RecordSpecMetadataAddress(contractSpecUUID, name), nil
}

// MustGetAsRecordSpecAddress returns the MetadataAddress for a record spec using the contract spec UUID within the current context and the provided name
// This is the same as AsRecordSpecAddress except it panics on error.
func (ma MetadataAddress) MustGetAsRecordSpecAddress(name string) MetadataAddress {
	retval, err := ma.AsRecordSpecAddress(name)
	if err != nil {
		panic(err)
	}
	return retval
}

// AsContractSpecAddress returns the MetadataAddress for a contract spec using the contract spec UUID within the current context
func (ma MetadataAddress) AsContractSpecAddress() (MetadataAddress, error) {
	contractSpecUUID, err := ma.ContractSpecUUID()
	if err != nil {
		return MetadataAddress{}, err
	}
	return ContractSpecMetadataAddress(contractSpecUUID), nil
}

// MustGetAsContractSpecAddress returns the MetadataAddress for a contract spec using the contract spec UUID within the current context
// This is the same as AsContractSpecAddress except it panics on error.
func (ma MetadataAddress) MustGetAsContractSpecAddress() MetadataAddress {
	retval, err := ma.AsContractSpecAddress()
	if err != nil {
		panic(err)
	}
	return retval
}

// ScopeSessionIteratorPrefix returns an iterator prefix that finds all Sessions assigned to the scope designated in this MetadataAddress.
// If the current address is empty this returns a prefix to iterate through all sessions.
// If the current address is a scope, this returns a prefix to iterate through all sessions in this scope.
// If the current address is a session or record, this returns a prefix to iterate through all sessions in the scope that contains
// this session or record.
func (ma MetadataAddress) ScopeSessionIteratorPrefix() ([]byte, error) {
	if len(ma) < 1 {
		return SessionKeyPrefix, nil
	}
	// if we don't know this type
	if !ma.isTypeOneOf(ScopeKeyPrefix, SessionKeyPrefix, RecordKeyPrefix) {
		return []byte{}, fmt.Errorf("this metadata address does not contain a scope uuid")
	}
	return append(SessionKeyPrefix, ma[1:17]...), nil
}

// ScopeRecordIteratorPrefix returns an iterator prefix that finds all Records assigned to the scope designated in this MetadataAddress.
// If the current address is empty this returns a prefix to iterate through all records.
// If the current address is a scope, this returns a prefix to iterate through all records in this scope.
// If the current address is a session or record, this returns a prefix to iterate through all records in the scope that contains
// this session or record.
func (ma MetadataAddress) ScopeRecordIteratorPrefix() ([]byte, error) {
	if len(ma) < 1 {
		return RecordKeyPrefix, nil
	}
	// if we don't know this type
	if !ma.isTypeOneOf(ScopeKeyPrefix, SessionKeyPrefix, RecordKeyPrefix) {
		return []byte{}, fmt.Errorf("this metadata address does not contain a scope uuid")
	}
	return append(RecordKeyPrefix, ma[1:17]...), nil
}

// ContractSpecRecordSpecIteratorPrefix returns an iterator prefix that finds all record specifications
// for the contract specification designated in this MetadataAddress.
// If the current address is empty this returns a prefix to iterate through all record specifications
// If the current address is a contract specification, this returns a prefix to iterate through all record specifications
// associated with that contract specification.
// If the current address is a record specification, this returns a prefix to iterate through all record specifications
// associated with the contract specification that contains this record specification.
// If the current address is some other type, an error is returned.
func (ma MetadataAddress) ContractSpecRecordSpecIteratorPrefix() ([]byte, error) {
	if len(ma) < 1 {
		return RecordSpecificationKeyPrefix, nil
	}
	// if we don't know this type
	if !ma.isTypeOneOf(ContractSpecificationKeyPrefix, RecordSpecificationKeyPrefix) {
		return []byte{}, fmt.Errorf("this metadata address does not contain a contract spec uuid")
	}
	return append(RecordSpecificationKeyPrefix, ma[1:17]...), nil
}

// Format implements fmt.Formatter interface for a MetadataAddress.
func (ma MetadataAddress) Format(s fmt.State, verb rune) {
	var out string
	switch verb {
	case 's', 'q':
		out = fmt.Sprintf(fmt.FormatString(s, verb), ma.String())
	case 'v':
		if s.Flag('#') {
			// We can't provide the same MetadataAddress arg back to Sprintf here (infinite recursion).
			// So we cast it as a byte slice, string that, then change the "[]byte" part to "MetadataAddress".
			out = fmt.Sprintf(fmt.FormatString(s, verb), []byte(ma))
			out = "MetadataAddress" + strings.TrimPrefix(out, "[]byte")
		} else {
			// The auto-generated gogoproto.stringer methods use "%v" for the MetadataAddress fields.
			// So here, we return the bech32 for "%v" so that MetadataAddress fields look right in those strings.
			out = fmt.Sprintf(fmt.FormatString(s, verb), ma.String())
		}
	default:
		// The other verbs (e.g. c b o O d x X U) should behave just like if this were a []byte.
		// Note that the 'p' (pointer) and 'T' (type) verbs are never processed using this Format method.
		// That's how %T returns the correct type even though it would actually return "[]byte" if run through this.
		out = fmt.Sprintf(fmt.FormatString(s, verb), []byte(ma))
	}

	_, err := s.Write([]byte(out))
	if err != nil {
		panic(err)
	}
}

// IsScopeAddress returns true if this address is valid and has a scope type byte.
func (ma MetadataAddress) IsScopeAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixScope)
}

// IsSessionAddress returns true if this address is valid and has a session type byte.
func (ma MetadataAddress) IsSessionAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixSession)
}

// IsRecordAddress returns true if the address is valid and has a record type byte.
func (ma MetadataAddress) IsRecordAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixRecord)
}

// IsScopeSpecificationAddress returns true if this address is valid and has a scope specification type byte.
func (ma MetadataAddress) IsScopeSpecificationAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixScopeSpecification)
}

// IsContractSpecificationAddress returns true if this address is valid and has a contract specification type byte.
func (ma MetadataAddress) IsContractSpecificationAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixContractSpecification)
}

// IsRecordSpecificationAddress returns true if this address is valid and has a record specification type byte.
func (ma MetadataAddress) IsRecordSpecificationAddress() bool {
	hrp, err := VerifyMetadataAddressFormat(ma)
	return (err == nil && hrp == PrefixRecordSpecification)
}

// isTypeOneOf returns true if the first byte is equal to the first byte in any provided options.
func (ma MetadataAddress) isTypeOneOf(options ...[]byte) bool {
	if len(ma) == 0 {
		return false
	}
	for _, o := range options {
		if len(o) > 0 && ma[0] == o[0] {
			return true
		}
	}
	return false
}

// ValidateIsScopeAddress returns an error if this isn't a valid MetadataAddress or if it's not a scope.
func (ma MetadataAddress) ValidateIsScopeAddress() error {
	return VerifyMetadataAddressHasType(ma, PrefixScope)
}

// ValidateIsScopeSpecificationAddress returns an error if this isn't a valid MetadataAddress or if it's not a scope specification.
func (ma MetadataAddress) ValidateIsScopeSpecificationAddress() error {
	return VerifyMetadataAddressHasType(ma, PrefixScopeSpecification)
}

// MetadataAddressDetails contains a breakdown of the components in a MetadataAddress.
type MetadataAddressDetails struct {
	// Address is the full MetadataAddress in question.
	Address MetadataAddress
	// AddressPrefix is the prefix bytes. It will have length 1 only if Address has a prefix portion.
	AddressPrefix []byte
	// AddressPrimaryUUID is the primary uuid. It will have length 16 only if Address has a primary UUID portion.
	AddressPrimaryUUID []byte
	// AddressSecondaryUUID is the secondary uuid. It will have length 16 only if Address has a secondary UUID portion.
	AddressSecondaryUUID []byte
	// AddressNameHash is the hashed name. It will have length 16 only if Address has a name hash portion.
	AddressNameHash []byte
	// AddressExcess is any bytes in Address that are not accounted for in the other byte arrays.
	AddressExcess []byte
	// Prefix is the human readable version of AddressPrefix. E.g. "scope"
	Prefix string
	// PrimaryUUID is the string version of AddressPrimaryUUID. E.g. "9e3e80f4-78ba-4fad-aed1-b79e1370cebd"
	PrimaryUUID string
	// SecondaryUUID is the string version of AddressSecondaryUUID. E.g. "164eb1bf-0818-4ad1-b3b9-39e86441d446"
	SecondaryUUID string
	// NameHashHex is the hex string encoded version of AddressNameHash. E.g. "787ec76dcafd20c1908eb0936a12f91e"
	NameHashHex string
	// NameHashBase64 is the base64 string encoded version of NameHashBase64. E.g. "eH7Hbcr9IMGQjrCTahL5Hg=="
	NameHashBase64 string
	// ExcessHex is the hex string encoded version of AddressExcess. E.g. "6578747261"
	ExcessHex string
	// ExcessBase64 is the base64 string encoded version of AddressExcess. E.g. "ZXh0cmE="
	ExcessBase64 string
	// ParentAddress is the MetadataAddress of the parent structure.
	// I.e. for session and record addresses, this will be a scope address.
	// For record spec addresses, this will be a contract spec address.
	// For all other types, it will be empty.
	ParentAddress MetadataAddress
}

// GetDetails breaks this MetadataAddress down into its various components, and returns all the details.
func (ma MetadataAddress) GetDetails() MetadataAddressDetails {
	// Copying this MetadataAddress to prevent weird behavior.
	addr := make(MetadataAddress, len(ma))
	copy(addr, ma)

	retval := MetadataAddressDetails{Address: addr}
	// Set the prefix info if we've got anything at all.
	if len(addr) >= 1 {
		retval.AddressPrefix = addr[0:1]
		prefix, err := addr.Prefix()
		if err != nil {
			// If there's an error, convert the prefix bytes to hex
			prefix = hex.EncodeToString(retval.AddressPrefix)
		}
		retval.Prefix = prefix
	}
	// Every type has a primary uuid as the 16 bytes after the prefix.
	// So if those exist, get set the primary uuid info.
	if len(addr) >= 17 {
		// Getting bytes directly so that we get the bytes even if .PrimaryUUID() would give an error.
		retval.AddressPrimaryUUID = addr[1:17]
		// Try to convert it to an actual UUID in order to use the UUID.String() method.
		// The only reason this conversion will fail is if the length isn't 16. We know it is, so just ignore the error.
		uid, _ := uuid.FromBytes(retval.AddressPrimaryUUID)
		retval.PrimaryUUID = uid.String()
	}
	// Secondary UUIDs or only for some types. Check if we've got one and set it accordingly.
	secondaryUUID, secondaryUUIDErr := addr.SecondaryUUID()
	if secondaryUUIDErr == nil {
		retval.AddressSecondaryUUID = secondaryUUID[:]
		retval.SecondaryUUID = secondaryUUID.String()
	}
	// Hashed names are only for some types. Check if we've got one and set it accordingly.
	nameHash, nameHashErr := addr.NameHash()
	if nameHashErr == nil {
		retval.AddressNameHash = nameHash
		retval.NameHashHex = hex.EncodeToString(retval.AddressNameHash)
		retval.NameHashBase64 = base64.StdEncoding.EncodeToString(retval.AddressNameHash)
	}
	// Check for any excess bytes
	expectedLength := 17 // 1 + 16 = prefix byte + primary UUID.
	if secondaryUUIDErr == nil || nameHashErr == nil {
		expectedLength += 16 // 16 = the secondary UUID length = the name hash length.
	}
	if len(addr) > expectedLength {
		retval.AddressExcess = addr[expectedLength:]
		retval.ExcessHex = hex.EncodeToString(retval.AddressExcess)
		retval.ExcessBase64 = base64.StdEncoding.EncodeToString(retval.AddressExcess)
	}
	// And set the parent if we can.
	if !addr.IsScopeAddress() {
		if pAddr, err := addr.AsScopeAddress(); err == nil {
			retval.ParentAddress = pAddr
		}
	}
	if !addr.IsContractSpecificationAddress() {
		if pAddr, err := addr.AsContractSpecAddress(); err == nil {
			retval.ParentAddress = pAddr
		}
	}
	return retval
}

// Denom gets the denom string for this MetadataAddress.
func (ma MetadataAddress) Denom() string {
	return DenomPrefix + ma.String()
}

// Coin creates the singular coin that represents this MetadataAddress.
func (ma MetadataAddress) Coin() sdk.Coin {
	return sdk.NewInt64Coin(ma.Denom(), 1)
}

// Coins creates a Coins with just the singular coin that represents this MetadataAddress.
func (ma MetadataAddress) Coins() sdk.Coins {
	return sdk.Coins{sdk.NewInt64Coin(ma.Denom(), 1)}
}

// AccMDLink associates an account address with a metadata address.
type AccMDLink struct {
	AccAddr sdk.AccAddress
	MDAddr  MetadataAddress
}

// NewAccMDLink creates a new association between an account address and a metadata address.
func NewAccMDLink(accAddr sdk.AccAddress, mdAddr MetadataAddress) *AccMDLink {
	return &AccMDLink{
		AccAddr: accAddr,
		MDAddr:  mdAddr,
	}
}

const (
	// nilStr is a string indicating that something is nil.
	nilStr = "<nil>"
	// emptyStr is a string indicating that something is empty.
	emptyStr = "<empty>"
)

// String returns a string representation of this AccMDLink in the format <AccAddr>:<MDAddr> using bech32 strings.
func (l *AccMDLink) String() string {
	if l == nil {
		return nilStr
	}
	return accStr(l.AccAddr) + ":" + mdStr(l.MDAddr)
}

// accStr returns a string representation of an AccAddress, either bech32 or a string indicating nil or empty.
func accStr(acc sdk.AccAddress) string {
	if len(acc) > 0 {
		return acc.String()
	}
	if acc == nil {
		return nilStr
	}
	return emptyStr
}

// accStr returns a string representation of a MetadataAddress, either bech32 or a string indicating nil or empty.
func mdStr(md MetadataAddress) string {
	if len(md) > 0 {
		return md.String()
	}
	if md == nil {
		return nilStr
	}
	return emptyStr
}

// AccMDLinks is a slice of AccMDLink.
type AccMDLinks []*AccMDLink

// String returns a string representation of this AccMDLinks.
func (a AccMDLinks) String() string {
	if len(a) > 0 {
		strs := make([]string, len(a))
		for i, link := range a {
			strs[i] = link.String()
		}
		return "[" + strings.Join(strs, ", ") + "]"
	}
	if a == nil {
		return nilStr
	}
	return emptyStr
}

// ValidateForScopes returns an error in the following cases:
//   - An entry is nil.
//   - An entry does not have an AccAddr.
//   - An entry does not have a MDAddr.
//   - An MDAddr is not a valid scope id.
//   - Any MDAddr appears more than once.
func (a AccMDLinks) ValidateForScopes() error {
	if len(a) == 0 {
		return nil
	}

	seenMDAddrs := make(map[string]int8)
	for _, link := range a {
		if link == nil {
			return errors.New("nil entry not allowed")
		}

		key := string(link.MDAddr)
		switch seenMDAddrs[key] {
		case 0:
			seenMDAddrs[key] = 1
			if err := link.MDAddr.ValidateIsScopeAddress(); err != nil {
				return err
			}
		case 1:
			seenMDAddrs[key] = 2
			return fmt.Errorf("duplicate metadata address %q not allowed", link.MDAddr)
		}

		if len(link.AccAddr) == 0 {
			return fmt.Errorf("no account address associated with metadata address %q", link.MDAddr)
		}
	}

	return nil
}

// GetAccAddrs returns a list of all AccAddr values from this AccMDLinks.
// Each entry will appear only once and in the order it first appears in this list.
// Empty and nil entries are ignored.
func (a AccMDLinks) GetAccAddrs() []sdk.AccAddress {
	seen := make(map[string]bool)
	var rv []sdk.AccAddress
	for _, link := range a {
		if link == nil || len(link.AccAddr) == 0 {
			continue
		}
		if !seen[string(link.AccAddr)] {
			seen[string(link.AccAddr)] = true
			rv = append(rv, link.AccAddr)
		}
	}
	return rv
}

// GetPrimaryUUIDs extracts the primary UUID out of each MDAddr and converts each to a string.
// If the MDAddr is nil, empty, or invalid, it's corresponding result will be "".
func (a AccMDLinks) GetPrimaryUUIDs() []string {
	if a == nil {
		return nil
	}
	rv := make([]string, len(a))
	for i, link := range a {
		if link != nil && len(link.MDAddr) > 0 {
			id, err := link.MDAddr.PrimaryUUID()
			if err == nil {
				rv[i] = id.String()
			}
		}
	}
	return rv
}

// GetMDAddrsForAccAddr returns all of the MDAddrs associated with the provided AccAddr.
func (a AccMDLinks) GetMDAddrsForAccAddr(addr string) []MetadataAddress {
	var rv []MetadataAddress
	for _, link := range a {
		if link != nil && addr == link.AccAddr.String() {
			rv = append(rv, link.MDAddr)
		}
	}
	return rv
}
