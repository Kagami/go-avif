package avif

import (
	"encoding/binary"
	"image"
	"io"
)

type fourCC [4]byte

var (
	boxTypeFTYP = fourCC{'f', 't', 'y', 'p'}
	boxTypeMDAT = fourCC{'m', 'd', 'a', 't'}
	boxTypeMETA = fourCC{'m', 'e', 't', 'a'}
	boxTypeHDLR = fourCC{'h', 'd', 'l', 'r'}
	boxTypePITM = fourCC{'p', 'i', 't', 'm'}
	boxTypeILOC = fourCC{'i', 'l', 'o', 'c'}
	boxTypeIINF = fourCC{'i', 'i', 'n', 'f'}
	boxTypeINFE = fourCC{'i', 'n', 'f', 'e'}
	boxTypeIPRP = fourCC{'i', 'p', 'r', 'p'}
	boxTypeIPCO = fourCC{'i', 'p', 'c', 'o'}
	boxTypeISPE = fourCC{'i', 's', 'p', 'e'}
	boxTypePASP = fourCC{'p', 'a', 's', 'p'}
	boxTypeAV1C = fourCC{'a', 'v', '1', 'C'}
	boxTypePIXI = fourCC{'p', 'i', 'x', 'i'}
	boxTypeIPMA = fourCC{'i', 'p', 'm', 'a'}

	itemTypeMIF1 = fourCC{'m', 'i', 'f', '1'}
	itemTypeAVIF = fourCC{'a', 'v', 'i', 'f'}
	itemTypeMIAF = fourCC{'m', 'i', 'a', 'f'}
	itemTypePICT = fourCC{'p', 'i', 'c', 't'}
	itemTypeMIME = fourCC{'m', 'i', 'm', 'e'}
	itemTypeURI  = fourCC{'u', 'r', 'i', ' '}
	itemTypeAV01 = fourCC{'a', 'v', '0', '1'}
)

func ulen(s string) uint32 {
	return uint32(len(s))
}

func bflag(b bool, pos uint8) uint8 {
	if b {
		return 1 << (pos - 1)
	}
	return 0
}

func writeAll(w io.Writer, writers ...io.WriterTo) (err error) {
	for _, wt := range writers {
		_, err = wt.WriteTo(w)
		if err != nil {
			return
		}
	}
	return
}

func writeBE(w io.Writer, chunks ...interface{}) (err error) {
	for _, v := range chunks {
		err = binary.Write(w, binary.BigEndian, v)
		if err != nil {
			return
		}
	}
	return
}

//----------------------------------------------------------------------

type box struct {
	size uint32
	typ  fourCC
}

func (b *box) Size() uint32 {
	return 8
}

func (b *box) WriteTo(w io.Writer) (n int64, err error) {
	err = writeBE(w, b.size, b.typ)
	return
}

//----------------------------------------------------------------------

type fullBox struct {
	box
	version uint8
	flags   uint32
}

func (b *fullBox) Size() uint32 {
	return 12
}

func (b *fullBox) WriteTo(w io.Writer) (n int64, err error) {
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	versionAndFlags := (uint32(b.version) << 24) | (b.flags & 0xffffff)
	err = writeBE(w, versionAndFlags)
	return
}

//----------------------------------------------------------------------

// File Type Box
type boxFTYP struct {
	box
	majorBrand       fourCC
	minorVersion     uint32
	compatibleBrands []fourCC
}

func (b *boxFTYP) Size() uint32 {
	return b.box.Size() +
		4 /*major_brand*/ + 4 /*minor_version*/ + uint32(len(b.compatibleBrands))*4
}

func (b *boxFTYP) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeFTYP
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.majorBrand, b.minorVersion, b.compatibleBrands)
	return
}

//----------------------------------------------------------------------

// Media Data Box
type boxMDAT struct {
	box
	data []byte
}

func (b *boxMDAT) Size() uint32 {
	return b.box.Size() + uint32(len(b.data))
}

func (b *boxMDAT) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeMDAT
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	_, err = w.Write(b.data)
	return
}

//----------------------------------------------------------------------

// The Meta box
type boxMETA struct {
	fullBox
	theHandler      boxHDLR
	primaryResource boxPITM
	itemLocations   boxILOC
	itemInfos       boxIINF
	itemProps       boxIPRP
}

func (b *boxMETA) Size() uint32 {
	return b.fullBox.Size() + b.theHandler.Size() + b.primaryResource.Size() +
		b.itemLocations.Size() + b.itemInfos.Size() + b.itemProps.Size()
}

func (b *boxMETA) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeMETA
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	err = writeAll(w, &b.theHandler, &b.primaryResource, &b.itemLocations,
		&b.itemInfos, &b.itemProps)
	return
}

//----------------------------------------------------------------------

// Handler Reference Box
type boxHDLR struct {
	fullBox
	preDefined  uint32
	handlerType fourCC
	reserved    [3]uint32
	name        string
}

func (b *boxHDLR) Size() uint32 {
	return b.fullBox.Size() +
		4 /*pre_defined*/ + 4 /*handler_type*/ + 12 /*reserved*/ +
		ulen(b.name) + 1 /*\0*/
}

func (b *boxHDLR) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeHDLR
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.preDefined, b.handlerType, b.reserved, []byte(b.name), []byte{0})
	return
}

//----------------------------------------------------------------------

// Primary Item Box
type boxPITM struct {
	fullBox
	itemID uint16
}

func (b *boxPITM) Size() uint32 {
	return b.fullBox.Size() + 2 /*item_ID*/
}

func (b *boxPITM) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypePITM
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.itemID)
	return
}

//----------------------------------------------------------------------

// The Item Location Box
type boxILOC struct {
	fullBox
	offsetSize     uint8 // 4 bits
	lengthSize     uint8 // 4 bits
	baseOffsetSize uint8 // 4 bits
	reserved       uint8 // 4 bits
	itemCount      uint16
	items          []boxILOCItem
}

func (b *boxILOC) Size() uint32 {
	size := b.fullBox.Size() + 1 /*offset_size + length_size*/ +
		1 /*base_offset_size + reserved*/ + 2 /*item_count*/
	for _, i := range b.items {
		size += 2 /*item_ID*/ + 2 /*data_reference_index*/ + uint32(b.baseOffsetSize) +
			2 /*extent_count*/ + uint32(len(i.extents))*uint32(b.offsetSize+b.lengthSize)
	}
	return size
}

func (b *boxILOC) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeILOC
	b.itemCount = uint16(len(b.items))
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	offsetSizeAndLengthSize := (b.offsetSize << 4) | (b.lengthSize & 0xf)
	baseOffsetSizeAndReserved := (b.baseOffsetSize << 4) | (b.reserved & 0xf)
	err = writeBE(w, offsetSizeAndLengthSize, baseOffsetSizeAndReserved, b.itemCount)
	if err != nil {
		return
	}
	for _, i := range b.items {
		err = i.write(w, b.baseOffsetSize, b.offsetSize, b.lengthSize)
		if err != nil {
			return
		}
	}
	return
}

type boxILOCItem struct {
	itemID             uint16
	dataReferenceIndex uint16
	baseOffset         uint64 // 0, 32 or 64 bits
	extentCount        uint16
	extents            []boxILOCItemExtent
}

func (i *boxILOCItem) write(w io.Writer, baseOffsetSize, offsetSize, lengthSize uint8) (err error) {
	i.extentCount = uint16(len(i.extents))
	var baseOffset interface{}
	baseOffset = []byte{}
	if baseOffsetSize == 4 {
		baseOffset = uint32(i.baseOffset)
	} else if baseOffsetSize == 8 {
		baseOffset = i.baseOffset
	}
	err = writeBE(w, i.itemID, i.dataReferenceIndex, baseOffset, i.extentCount)
	if err != nil {
		return
	}
	for _, e := range i.extents {
		if err = e.write(w, offsetSize, lengthSize); err != nil {
			return
		}
	}
	return
}

type boxILOCItemExtent struct {
	extentOffset uint64 // 0, 32 or 64 bits
	extentLength uint64 // 0, 32 or 64 bits
}

func (e *boxILOCItemExtent) write(w io.Writer, offsetSize, lengthSize uint8) (err error) {
	var extentOffset interface{}
	extentOffset = []byte{}
	if offsetSize == 4 {
		extentOffset = uint32(e.extentOffset)
	} else if offsetSize == 8 {
		extentOffset = e.extentOffset
	}
	var extentLength interface{}
	extentLength = []byte{}
	if lengthSize == 4 {
		extentLength = uint32(e.extentLength)
	} else if lengthSize == 8 {
		extentLength = e.extentLength
	}
	err = writeBE(w, extentOffset, extentLength)
	return
}

//----------------------------------------------------------------------

// Item Information Box
type boxIINF struct {
	fullBox
	entryCount uint16
	itemInfos  []boxINFEv2
}

func (b *boxIINF) Size() uint32 {
	size := b.fullBox.Size() + 2 /*entry_count*/
	for _, ie := range b.itemInfos {
		size += ie.Size()
	}
	return size
}

func (b *boxIINF) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeIINF
	b.entryCount = uint16(len(b.itemInfos))
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	if err = writeBE(w, b.entryCount); err != nil {
		return
	}
	for _, ie := range b.itemInfos {
		if _, err = ie.WriteTo(w); err != nil {
			return
		}
	}
	return
}

//----------------------------------------------------------------------

// Item Info Entry Box
type boxINFEv2 struct {
	fullBox
	itemID              uint16
	itemProtectionIndex uint16
	itemType            fourCC
	itemName            string
	contentType         string
	contentEncoding     string
	itemURIType         string
}

func (b *boxINFEv2) Size() uint32 {
	size := b.fullBox.Size() + 2 /*item_ID*/ + 2 /*item_protection_index*/ +
		4 /*item_type*/ + ulen(b.itemName) + 1 /*\0*/
	if b.itemType == itemTypeMIME {
		size += ulen(b.contentType) + 1 /*\0*/ + ulen(b.contentEncoding) + 1 /*\0*/
	} else if b.itemType == itemTypeURI {
		size += ulen(b.itemURIType)
	}
	return size
}

func (b *boxINFEv2) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeINFE
	b.version = 2
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.itemID, b.itemProtectionIndex, b.itemType,
		[]byte(b.itemName), []byte{0})
	if err != nil {
		return
	}
	if b.itemType == itemTypeMIME {
		// XXX(Kagami): Skip content_encoding if it's empty?
		err = writeBE(w, []byte(b.contentType), []byte{0}, []byte(b.contentEncoding), []byte{0})
	} else if b.itemType == itemTypeURI {
		// XXX(Kagami): Shouldn't be null-terminated per spec?
		err = writeBE(w, []byte(b.itemURIType))
	}
	return
}

//----------------------------------------------------------------------

// Item Properties Box
type boxIPRP struct {
	box
	propertyContainer boxIPCO
	association       boxIPMA
}

func (b *boxIPRP) Size() uint32 {
	return b.box.Size() + b.propertyContainer.Size() + b.association.Size()
}

func (b *boxIPRP) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeIPRP
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	err = writeAll(w, &b.propertyContainer, &b.association)
	return
}

//----------------------------------------------------------------------

// Item Property Container Box
type boxIPCO struct {
	box
	properties []boxIPCOProperty
}

func (b *boxIPCO) Size() uint32 {
	size := b.box.Size()
	for _, p := range b.properties {
		size += p.Size()
	}
	return size
}

func (b *boxIPCO) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeIPCO
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	for _, p := range b.properties {
		if _, err = p.WriteTo(w); err != nil {
			return
		}
	}
	return
}

type boxIPCOProperty interface {
	io.WriterTo
	Size() uint32
}

//----------------------------------------------------------------------

// Image spatial extents
type boxISPE struct {
	fullBox
	imageWidth  uint32
	imageHeight uint32
}

func (b *boxISPE) Size() uint32 {
	return b.fullBox.Size() + 4 /*image_width*/ + 4 /*image_height*/
}

func (b *boxISPE) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeISPE
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.imageWidth, b.imageHeight)
	return
}

//----------------------------------------------------------------------

// Pixel aspect ratio
type boxPASP struct {
	box
	hSpacing uint32
	vSpacing uint32
}

func (b *boxPASP) Size() uint32 {
	return b.box.Size() + 4 /*hSpacing*/ + 4 /*vSpacing*/
}

func (b *boxPASP) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypePASP
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	err = writeBE(w, b.hSpacing, b.vSpacing)
	return
}

//----------------------------------------------------------------------

// Pixel aspect ratio
type boxAV1C struct {
	box
	av1Config boxAV1CConfig
}

func (b *boxAV1C) Size() uint32 {
	return b.box.Size() + 1 /*marker + version*/ + 1 /*seq_profile + seq_level_idx_0*/ +
		// seq_tier_0 + high_bitdepth + twelve_bit + monochrome +
		// chroma_subsampling_x + chroma_subsampling_y + chroma_sample_position
		1 +
		// reserved + initial_presentation_delay_present + initial_presentation_delay_minus_one/reserved
		1 +
		uint32(len(b.av1Config.configOBUs))
}

func (b *boxAV1C) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeAV1C
	if _, err = b.box.WriteTo(w); err != nil {
		return
	}
	err = b.av1Config.write(w)
	return
}

type boxAV1CConfig struct {
	marker  bool
	version uint8 // 7 bits
	//---
	seqProfile   uint8 // 4 bits
	seqLevelIdx0 uint8 // 4 bits
	//---
	seqTier0             bool
	highBitdepth         bool
	twelveBit            bool
	monochrome           bool
	chromaSubsamplingX   bool
	chromaSubsamplingY   bool
	chromaSamplePosition uint8 // 2 bits
	//---
	reserved                         uint8 // 3 bits
	initialPresentationDelayPresent  bool
	initialPresentationDelayMinusOne uint8 // 4 bits
	reserved2                        uint8 // 4 bits
	//---
	configOBUs []byte
}

func (c *boxAV1CConfig) write(w io.Writer) (err error) {
	c.marker = true
	c.version = 1
	c.reserved = 0
	c.reserved2 = 0
	markerAndVersion := bflag(c.marker, 8) | (c.version & 0x7f)
	seqProfileAndSeqLevelIdx0 := (c.seqProfile << 5) | (c.seqLevelIdx0 & 0x1f)
	codecParams := bflag(c.seqTier0, 8) |
		bflag(c.highBitdepth, 7) |
		bflag(c.twelveBit, 6) |
		bflag(c.monochrome, 5) |
		bflag(c.chromaSubsamplingX, 4) |
		bflag(c.chromaSubsamplingY, 3) |
		(c.chromaSamplePosition & 3)
	presentationParams := (c.reserved << 5) | bflag(c.initialPresentationDelayPresent, 4)
	if c.initialPresentationDelayPresent {
		presentationParams |= c.initialPresentationDelayMinusOne & 0xf
	} else {
		presentationParams |= c.reserved2 & 0xf
	}
	err = writeBE(w, markerAndVersion, seqProfileAndSeqLevelIdx0, codecParams,
		presentationParams)
	if err != nil {
		return
	}
	_, err = w.Write(c.configOBUs)
	return
}

//----------------------------------------------------------------------

// Pixel information
type boxPIXI struct {
	fullBox
	numChannels    uint8
	bitsPerChannel []uint8
}

func (b *boxPIXI) Size() uint32 {
	return b.fullBox.Size() + 1 /*num_channels*/ + uint32(len(b.bitsPerChannel))
}

func (b *boxPIXI) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypePIXI
	b.numChannels = uint8(len(b.bitsPerChannel))
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	if err = writeBE(w, b.numChannels); err != nil {
		return
	}
	for _, bpc := range b.bitsPerChannel {
		if err = writeBE(w, bpc); err != nil {
			return
		}
	}
	return
}

//----------------------------------------------------------------------

// Item Property Association
type boxIPMA struct {
	fullBox
	entryCount uint32
	entries    []boxIPMAAssociation
}

func (b *boxIPMA) Size() uint32 {
	propSize := 1
	if b.flags&1 == 1 {
		propSize = 2
	}
	size := b.fullBox.Size() + 4 /*entry_count*/
	for _, a := range b.entries {
		size += 2 /*item_ID*/ + 1 /*association_count*/ +
			uint32(len(a.props))*uint32(propSize)
	}
	return size
}

func (b *boxIPMA) WriteTo(w io.Writer) (n int64, err error) {
	b.size = b.Size()
	b.typ = boxTypeIPMA
	b.entryCount = uint32(len(b.entries))
	if _, err = b.fullBox.WriteTo(w); err != nil {
		return
	}
	if err = writeBE(w, b.entryCount); err != nil {
		return
	}
	for _, a := range b.entries {
		if err = a.write(w, b.flags); err != nil {
			return
		}
	}
	return
}

type boxIPMAAssociation struct {
	itemID           uint16
	associationCount uint8
	props            []boxIPMAAssociationProperty
}

func (a *boxIPMAAssociation) write(w io.Writer, flags uint32) (err error) {
	// TODO(Kagami): Make sure slice length isn't overflowed?
	a.associationCount = uint8(len(a.props))
	if err = writeBE(w, a.itemID, a.associationCount); err != nil {
		return
	}
	for _, p := range a.props {
		if err = p.write(w, flags); err != nil {
			return
		}
	}
	return
}

type boxIPMAAssociationProperty struct {
	essential     bool
	propertyIndex uint16 // 7 or 15 bits
}

func (p *boxIPMAAssociationProperty) write(w io.Writer, flags uint32) (err error) {
	essential := 0
	if p.essential {
		essential = 1
	}
	if flags&1 == 1 {
		v := (p.propertyIndex & 0x7fff) | uint16(essential<<15)
		err = writeBE(w, v)
	} else {
		v := uint8(p.propertyIndex&0x7f) | uint8(essential<<7)
		err = writeBE(w, v)
	}
	return
}

//----------------------------------------------------------------------

func getSubsamplingXY(subsampling image.YCbCrSubsampleRatio) (x bool, y bool) {
	switch subsampling {
	case image.YCbCrSubsampleRatio420:
		return true, true
	case image.YCbCrSubsampleRatio422:
		return true, false
	case image.YCbCrSubsampleRatio444:
		return false, false
	}
	return
}

func muxFrame(w io.Writer, m image.Image, subsampling image.YCbCrSubsampleRatio, obuData []byte) (err error) {
	// TODO(Kagami): Parse params from Sequence Header OBU instead?
	rec := m.Bounds()
	width := uint32(rec.Max.X - rec.Min.X)
	height := uint32(rec.Max.Y - rec.Min.Y)
	sx, sy := getSubsamplingXY(subsampling)

	fileData := boxMDAT{data: obuData}
	fileType := boxFTYP{
		majorBrand:       itemTypeMIF1,
		compatibleBrands: []fourCC{itemTypeMIF1, itemTypeAVIF, itemTypeMIAF},
	}
	metadata := boxMETA{
		theHandler: boxHDLR{
			handlerType: itemTypePICT,
			name:        "go-avif v0",
		},
		primaryResource: boxPITM{itemID: 1},
		itemLocations: boxILOC{
			// NOTE(Kagami): We predefine location item even while we don't
			// know corrent offsets yet in order to fix them in place later.
			// It's needed because meta box goes before mdat box therefore
			// size of the metadata can't change. We only use baseOffset and
			// extentLength so occupy 32-bit storage space for them. They're
			// unlikely to overflow (>4GB image is not practical).
			lengthSize:     4,
			baseOffsetSize: 4,
			items: []boxILOCItem{
				{
					itemID:  1,
					extents: []boxILOCItemExtent{{}},
				},
			},
		},
		itemInfos: boxIINF{
			itemInfos: []boxINFEv2{
				{
					itemID:   1,
					itemType: itemTypeAV01,
					itemName: "Image",
				},
			},
		},
		itemProps: boxIPRP{
			propertyContainer: boxIPCO{
				properties: []boxIPCOProperty{
					&boxISPE{imageWidth: width, imageHeight: height},
					&boxPASP{hSpacing: 1, vSpacing: 1},
					&boxAV1C{
						// Only 8-bit at the moment.
						av1Config: boxAV1CConfig{
							chromaSubsamplingX: sx,
							chromaSubsamplingY: sy,
						},
					},
					&boxPIXI{bitsPerChannel: []uint8{8, 8, 8}},
				},
			},
			association: boxIPMA{
				entries: []boxIPMAAssociation{
					{
						itemID: 1,
						props: []boxIPMAAssociationProperty{
							{false, 1}, // non-essential width/height
							{false, 2}, // non-essential aspect ratio
							{true, 3},  // essential AV1 config
							{true, 4},  // essential bitdepth
						},
					},
				},
			},
		},
	}
	// Can fix iloc offsets now.
	locItem := &metadata.itemLocations.items[0]
	locItem.baseOffset = uint64(fileType.Size() + metadata.Size() + fileData.box.Size())
	locItem.extents[0].extentLength = uint64(len(fileData.data))

	err = writeAll(w, &fileType, &metadata, &fileData)
	return
}
