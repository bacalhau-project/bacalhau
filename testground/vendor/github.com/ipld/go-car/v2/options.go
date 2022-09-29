package car

import (
	"math"

	"github.com/ipld/go-car/v2/index"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/multiformats/go-multicodec"

	"github.com/ipld/go-car/v2/internal/carv1"
)

// DefaultMaxIndexCidSize specifies the maximum size in byptes accepted as a section CID by CARv2 index.
const DefaultMaxIndexCidSize = 2 << 10 // 2 KiB

// DefaultMaxAllowedHeaderSize specifies the default maximum size that a CARv1
// decode (including within a CARv2 container) will allow a header to be without
// erroring. This is to prevent OOM errors where a header prefix includes a
// too-large size specifier.
// Currently set to 32 MiB.
const DefaultMaxAllowedHeaderSize = carv1.DefaultMaxAllowedHeaderSize

// DefaultMaxAllowedHeaderSize specifies the default maximum size that a CARv1
// decode (including within a CARv2 container) will allow a section to be
// without erroring. This is to prevent OOM errors where a section prefix
// includes a too-large size specifier.
// Typically IPLD blocks should be under 2 MiB (ideally under 1 MiB), so unless
// atypical data is expected, this should not be a large value.
// Currently set to 8 MiB.
const DefaultMaxAllowedSectionSize = carv1.DefaultMaxAllowedSectionSize

// Option describes an option which affects behavior when interacting with CAR files.
type Option func(*Options)

// ReadOption hints that an API wants options related only to reading CAR files.
type ReadOption = Option

// WriteOption hints that an API wants options related only to reading CAR files.
type WriteOption = Option

// ReadWriteOption is either a ReadOption or a WriteOption.
// Deprecated: use Option instead.
type ReadWriteOption = Option

// Options holds the configured options after applying a number of
// Option funcs.
//
// This type should not be used directly by end users; it's only exposed as a
// side effect of Option.
type Options struct {
	DataPadding            uint64
	IndexPadding           uint64
	IndexCodec             multicodec.Code
	ZeroLengthSectionAsEOF bool
	MaxIndexCidSize        uint64
	StoreIdentityCIDs      bool

	BlockstoreAllowDuplicatePuts bool
	BlockstoreUseWholeCIDs       bool
	MaxTraversalLinks            uint64
	WriteAsCarV1                 bool
	TraversalPrototypeChooser    traversal.LinkTargetNodePrototypeChooser

	MaxAllowedHeaderSize  uint64
	MaxAllowedSectionSize uint64
}

// ApplyOptions applies given opts and returns the resulting Options.
// This function should not be used directly by end users; it's only exposed as a
// side effect of Option.
func ApplyOptions(opt ...Option) Options {
	opts := Options{
		MaxTraversalLinks:     math.MaxInt64, //default: traverse all
		MaxAllowedHeaderSize:  carv1.DefaultMaxAllowedHeaderSize,
		MaxAllowedSectionSize: carv1.DefaultMaxAllowedSectionSize,
	}
	for _, o := range opt {
		o(&opts)
	}
	// Set defaults for zero valued fields.
	if opts.IndexCodec == 0 {
		opts.IndexCodec = multicodec.CarMultihashIndexSorted
	}
	if opts.MaxIndexCidSize == 0 {
		opts.MaxIndexCidSize = DefaultMaxIndexCidSize
	}
	return opts
}

// ZeroLengthSectionAsEOF sets whether to allow the CARv1 decoder to treat
// a zero-length section as the end of the input CAR file. For example, this can
// be useful to allow "null padding" after a CARv1 without knowing where the
// padding begins.
func ZeroLengthSectionAsEOF(enable bool) Option {
	return func(o *Options) {
		o.ZeroLengthSectionAsEOF = enable
	}
}

// UseDataPadding sets the padding to be added between CARv2 header and its data payload on Finalize.
func UseDataPadding(p uint64) Option {
	return func(o *Options) {
		o.DataPadding = p
	}
}

// UseIndexPadding sets the padding between data payload and its index on Finalize.
func UseIndexPadding(p uint64) Option {
	return func(o *Options) {
		o.IndexPadding = p
	}
}

// UseIndexCodec sets the codec used for index generation.
func UseIndexCodec(c multicodec.Code) Option {
	return func(o *Options) {
		o.IndexCodec = c
	}
}

// WithoutIndex flags that no index should be included in generation.
func WithoutIndex() Option {
	return func(o *Options) {
		o.IndexCodec = index.CarIndexNone
	}
}

// StoreIdentityCIDs sets whether to persist sections that are referenced by
// CIDs with multihash.IDENTITY digest.
// When writing CAR files with this option,
// Characteristics.IsFullyIndexed will be set.
//
// This option is disabled by default.
func StoreIdentityCIDs(b bool) Option {
	return func(o *Options) {
		o.StoreIdentityCIDs = b
	}
}

// MaxIndexCidSize specifies the maximum allowed size for indexed CIDs in bytes.
// Indexing a CID with larger than the allowed size results in ErrCidTooLarge error.
func MaxIndexCidSize(s uint64) Option {
	return func(o *Options) {
		o.MaxIndexCidSize = s
	}
}

// WithTraversalPrototypeChooser specifies the prototype chooser that should be used
// when performing traversals in writes from a linksystem.
func WithTraversalPrototypeChooser(t traversal.LinkTargetNodePrototypeChooser) Option {
	return func(o *Options) {
		o.TraversalPrototypeChooser = t
	}
}

// MaxAllowedHeaderSize overrides the default maximum size (of 32 MiB) that a
// CARv1 decode (including within a CARv2 container) will allow a header to be
// without erroring.
func MaxAllowedHeaderSize(max uint64) Option {
	return func(o *Options) {
		o.MaxAllowedHeaderSize = max
	}
}

// MaxAllowedSectionSize overrides the default maximum size (of 8 MiB) that a
// CARv1 decode (including within a CARv2 container) will allow a header to be
// without erroring.
// Typically IPLD blocks should be under 2 MiB (ideally under 1 MiB), so unless
// atypical data is expected, this should not be a large value.
func MaxAllowedSectionSize(max uint64) Option {
	return func(o *Options) {
		o.MaxAllowedSectionSize = max
	}
}
