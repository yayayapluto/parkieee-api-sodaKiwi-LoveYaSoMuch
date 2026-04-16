package zone

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	zones          []Zone
	pairingRows    map[string]*GatePairingCode
	confirmInvoked bool
	lastGateToken  string
}

func (f *fakeRepo) ListZones() ([]Zone, error) { return f.zones, nil }
func (f *fakeRepo) CreateZone(z *Zone) error {
	f.zones = append(f.zones, *z)
	return nil
}
func (f *fakeRepo) FindZoneByID(uuid.UUID) (*Zone, error)                         { return nil, nil }
func (f *fakeRepo) UpdateZone(*Zone) error                                        { return nil }
func (f *fakeRepo) ListGates() ([]Gate, error)                                    { return nil, nil }
func (f *fakeRepo) CreateGate(*Gate) error                                        { return nil }
func (f *fakeRepo) FindGateByID(uuid.UUID) (*Gate, error)                         { return nil, nil }
func (f *fakeRepo) UpdateGate(*Gate) error                                        { return nil }
func (f *fakeRepo) ListGateDevices(uuid.UUID) ([]GateDevice, error)               { return nil, nil }
func (f *fakeRepo) UpdateGateDeviceStatus(uuid.UUID, string, string) error        { return nil }
func (f *fakeRepo) CreatePairingCode(pc *GatePairingCode) error {
	if f.pairingRows == nil {
		f.pairingRows = map[string]*GatePairingCode{}
	}
	f.pairingRows[pc.Code] = pc
	return nil
}
func (f *fakeRepo) FindActivePairingCode(code string, now time.Time) (*GatePairingCode, error) {
	row := f.pairingRows[code]
	if row == nil || !row.ExpiresAt.After(now) {
		return nil, nil
	}
	return row, nil
}
func (f *fakeRepo) ConfirmPairingCode(code string, gateID, _ uuid.UUID, gateToken string, now time.Time) error {
	row := f.pairingRows[code]
	if row == nil {
		return nil
	}
	f.confirmInvoked = true
	f.lastGateToken = gateToken
	row.Status = PairingStatusConfirmed
	row.GateID = &gateID
	row.GateJWT = gateToken
	confirmedAt := now
	row.ConfirmedAt = &confirmedAt
	return nil
}

func TestNewService(t *testing.T) {
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("expected service instance")
	}
}

func TestService_CreateZone(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	zone, err := svc.CreateZone("Zone A", "Main", 20, 1000, nil)
	require.NoError(t, err)
	require.NotNil(t, zone)
	assert.Equal(t, "Zone A", zone.Name)
	assert.Len(t, repo.zones, 1)
}

func TestService_PairingFlow(t *testing.T) {
	repo := &fakeRepo{pairingRows: map[string]*GatePairingCode{}}
	svc := NewService(repo)
	now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	request, err := svc.RequestPairingCode()
	require.NoError(t, err)
	require.NotNil(t, request)
	assert.Len(t, request.Code, 6)

	gateID := uuid.New()
	err = svc.ConfirmPairingCode(request.Code, gateID, uuid.New())
	require.NoError(t, err)
	assert.True(t, repo.confirmInvoked)
	assert.NotEmpty(t, repo.lastGateToken)

	verified, err := svc.VerifyPairingCode(request.Code)
	require.NoError(t, err)
	require.NotNil(t, verified)
	assert.Equal(t, PairingStatusConfirmed, verified.Status)
	require.NotNil(t, verified.GateID)
	assert.Equal(t, gateID, *verified.GateID)
	assert.NotEmpty(t, verified.GateToken)
}
