package vectors_test

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"testing"

	"github.com/NTARI-RAND/sohocloud-protocol/canon"
	"github.com/NTARI-RAND/sohocloud-protocol/employment"
	"github.com/NTARI-RAND/sohocloud-protocol/listing"
)

// These signatures were generated and verified by the actual v0.1.0 module.
// Earlier releases changed listing/assignment bytes, so upgrading consumers
// must not assume that old nodes and new coordinators can interoperate.
func TestLegacySignaturesRequireCoordinatedMigration(t *testing.T) {
	data, err := os.ReadFile("../testdata/legacy_v0_1_signatures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		PublicKey  ed25519.PublicKey
		Listing    listing.CapabilityListing
		Assignment employment.Assignment
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.PublicKey) != ed25519.PublicKeySize {
		t.Fatal("invalid legacy fixture public key")
	}
	if !ed25519.Verify(fixture.PublicKey, legacyListingBytes(fixture.Listing), fixture.Listing.Signature) {
		t.Fatal("fixture listing must verify under v0.1.0")
	}
	if !ed25519.Verify(fixture.PublicKey, legacyAssignmentBytes(fixture.Assignment), fixture.Assignment.Signature) {
		t.Fatal("fixture assignment must verify under v0.1.0")
	}
	if fixture.Listing.Verify(fixture.PublicKey) {
		t.Fatal("legacy listing unexpectedly verified; reassess migration boundary")
	}
	if fixture.Assignment.Verify(fixture.PublicKey) {
		t.Fatal("legacy assignment unexpectedly verified; reassess migration boundary")
	}
}

// Reconstruct the v0.1.0 fields in their original order, before GPU and storage
// fields were added to either message.
func legacyListingBytes(advert listing.CapabilityListing) []byte {
	b := canon.New("sohocloud/listing/v0")
	b.String(string(advert.NodeID))
	b.String(string(advert.Class))
	b.Count(len(advert.Printers))
	for _, printer := range advert.Printers {
		b.String(string(printer.Kind))
		b.String(printer.Model)
	}
	b.Int64(int64(advert.Capacity.VCPUs))
	b.Int64(int64(advert.Capacity.MemMB))
	b.Int64(int64(advert.Capacity.DiskMB))
	b.Int64(int64(advert.Capacity.PrintQPS))
	b.Bool(advert.OptIn.Compute)
	b.Bool(advert.OptIn.Print)
	b.Time(advert.IssuedAt)
	b.Uint64(advert.Seq)
	return b.Sum()
}

func legacyAssignmentBytes(assignment employment.Assignment) []byte {
	b := canon.New("sohocloud/assignment/v0")
	b.String(assignment.JobID)
	b.String(string(assignment.NodeID))
	b.String(assignment.Spec.Workload)
	b.String(assignment.Spec.Image)
	b.Count(len(assignment.Spec.Args))
	for _, arg := range assignment.Spec.Args {
		b.String(arg)
	}
	b.String(assignment.Spec.PrinterKind)
	b.Int64(int64(assignment.Fee.ContributorShareBps))
	b.Int64(int64(assignment.Fee.PlatformFeeBps))
	b.Time(assignment.OfferedAt)
	return b.Sum()
}
