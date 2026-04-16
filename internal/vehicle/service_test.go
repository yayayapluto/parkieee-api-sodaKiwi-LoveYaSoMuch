package vehicle_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	vehiclemocks "github.com/yyypluto/parkieee-api/internal/vehicle/mocks"
)

func TestAssignRFID_DuplicateActiveCard_ReturnsError(t *testing.T) {
	repo := vehiclemocks.NewRepo(t)
	svc := vehicle.NewService(repo)

	vehicleID := uuid.New()
	repo.On("FindVehicleByID", vehicleID).Return(&vehicle.Vehicle{ID: vehicleID}, nil)
	repo.On("FindRFIDByCardUID", "CARD-001").Return(&vehicle.RFIDCard{ID: uuid.New(), CardUID: "CARD-001", IsActive: true}, nil)

	_, err := svc.AssignRFID(vehicleID, "CARD-001")

	assert.ErrorIs(t, err, vehicle.ErrCardAlreadyAssigned)
	repo.AssertNotCalled(t, "CreateRFIDCard", mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeleteVehicleType_InUse_ReturnsError(t *testing.T) {
	repo := vehiclemocks.NewRepo(t)
	svc := vehicle.NewService(repo)

	vehicleTypeID := uuid.New()
	repo.On("FindVehicleTypeByID", vehicleTypeID).Return(&vehicle.VehicleType{ID: vehicleTypeID}, nil)
	repo.On("VehicleTypeInUse", vehicleTypeID).Return(true, nil)

	err := svc.DeleteVehicleType(vehicleTypeID)

	assert.ErrorIs(t, err, vehicle.ErrInUse)
	repo.AssertNotCalled(t, "DeleteVehicleType", mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeactivateRFID_SetsDeactivatedByAndAt(t *testing.T) {
	repo := vehiclemocks.NewRepo(t)
	svc := vehicle.NewService(repo)

	cardID := uuid.New()
	actorID := uuid.New()
	repo.On("FindRFIDByID", cardID).Return(&vehicle.RFIDCard{ID: cardID, CardUID: "CARD-001", IsActive: true}, nil)
	repo.On("DeactivateRFID", cardID, actorID, mock.AnythingOfType("time.Time")).Run(func(args mock.Arguments) {
		deactivatedAt := args.Get(2).(time.Time)
		assert.False(t, deactivatedAt.IsZero())
	}).Return(nil)

	err := svc.DeactivateRFID(cardID, actorID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
