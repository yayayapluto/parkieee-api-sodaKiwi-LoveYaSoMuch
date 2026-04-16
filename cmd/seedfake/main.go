// cmd/seedfake/main.go
//
// Bulk seed massive fake data across all models via go-faker.
// Truncates all domain data tables before re-seeding to guarantee clean states.
// Usage:
//   go run ./cmd/seedfake               # default scale 1 (1000 txs)
//   go run ./cmd/seedfake -scale=2      # 2000 txs
//   go run ./cmd/seedfake -password=lol # custom shared password (default: password123)
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/yyypluto/parkieee-api/config"
	"github.com/yyypluto/parkieee-api/internal/auth"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"github.com/yyypluto/parkieee-api/internal/payment"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/internal/user"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	"github.com/yyypluto/parkieee-api/internal/zone"
	"github.com/yyypluto/parkieee-api/pkg/database"
)

var (
	sillyFirstNames = []string{
		"Bambang", "Joko", "Asep", "Udin", "Tukiyem", "Sumiati",
		"Slamet", "Paijo", "Sukijan", "Wati", "Tejo", "Gatot",
		"Dudung", "Mamat", "Kusno", "Marno", "Yanto", "Markonah",
	}
	sillyAdjectives = []string{
		"Galak", "Ngantuk", "Lapar", "Cepat", "Lemot", "Santuy",
		"Ganteng", "Lugu", "Sangar", "Ceria", "Mager", "Halu",
	}
	sillyAnimals = []string{
		"Kucing", "Bebek", "Tikus", "Gajah", "Kambing", "Tongkol",
	}
	sillyFoods = []string{
		"Bakso", "Mie", "Sate", "Rendang", "Soto", "Gudeg",
	}
)

func main() {
	scaleOpt := flag.Int("scale", 1, "scale multiplier for transactions")
	passwordOpt := flag.String("password", "password123", "shared plaintext password")
	flag.Parse()

	if *scaleOpt <= 0 {
		log.Fatalf("scale must be > 0, got %d", *scaleOpt)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	tenantID := mustUUID(cfg.TenantID)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Resolve auth roles created by cmd/seed
	roleIDs, err := loadRoleIDs(db)
	if err != nil {
		log.Fatalf("load roles failed: %v (run go run ./cmd/seed first)", err)
	}

	// 1. Truncate
	log.Println("[1/7] Truncating domain tables...")
	truncateDomainTables(db)

	adminUserID := getAdminID(db)

	// 2. Users
	log.Println("[2/7] Seeding Users...")
	hash, _ := bcrypt.GenerateFromPassword([]byte(*passwordOpt), bcrypt.DefaultCost)
	users := seedUsers(db, rng, string(hash), tenantID, roleIDs, 50**scaleOpt)

	// 3. Vehicle Types
	log.Println("[3/7] Seeding Vehicle Types...")
	vTypes := seedVehicleTypes(db, rng)

	// 4. Zones & Gates
	log.Println("[4/7] Seeding Zones & Gates...")
	zones, entryGates, exitGates := seedZonesAndGates(db, rng, tenantID, &adminUserID)

	// 5. Fee Configs
	log.Println("[5/7] Seeding Fee Configs & Tiers...")
	feeConfigs := seedFeeConfigs(db, rng, zones, vTypes, tenantID)

	// 6. Vehicles & RFID Cards
	log.Println("[6/7] Seeding Vehicles & RFID Cards...")
	vehicles, rfidCards := seedVehicles(db, rng, vTypes, tenantID, 200**scaleOpt)

	// 7. Transactions & Payments
	log.Println("[7/7] Seeding Transactions & Payments...")
	seedTransactions(db, rng, tenantID, entryGates, exitGates, vehicles, rfidCards, feeConfigs, users, 1000**scaleOpt)

	log.Printf("done. fake data massively generated with scale %d.", *scaleOpt)
}

func truncateDomainTables(db *gorm.DB) {
	// Exclude Users first because we need to delete only fake users, not admin.
	db.Exec(`DELETE FROM users WHERE username != 'admin'`)

	tables, err := db.Migrator().GetTables()
	if err != nil {
		log.Fatalf("get tables failed: %v", err)
	}

	var toTruncate []string
	for _, t := range tables {
		if t == "roles" || t == "permissions" || t == "role_permissions" || t == "system_configs" || t == "users" || t == "migrations" {
			continue
		}
		toTruncate = append(toTruncate, t)
	}

	if len(toTruncate) > 0 {
		cmd := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(toTruncate, ", "))
		if err := db.Exec(cmd).Error; err != nil {
			log.Fatalf("truncate failed: %v", err)
		}
	}
}

func getAdminID(db *gorm.DB) uuid.UUID {
	var admin user.User
	db.Where("username = ?", "admin").First(&admin)
	if admin.ID == uuid.Nil {
		// fallback dummy if no admin
		return uuid.New()
	}
	return admin.ID
}

// -------------------------------------------------------------
// SEEDERS
// -------------------------------------------------------------

func seedUsers(db *gorm.DB, rng *rand.Rand, hash string, tenantID uuid.UUID, roleIDs map[string]uuid.UUID, count int) []auth.User {
	var out []auth.User
	for i := 0; i < count; i++ {
		silly := pick(rng, sillyFirstNames)
		adj := pick(rng, sillyAdjectives)
		thing := pick(rng, append(sillyAnimals, sillyFoods...))
		realName := faker.FirstName() + " " + faker.LastName()
		displayName := fmt.Sprintf("%s %s %s (%s)", silly, adj, thing, realName)

		username := strings.ToLower(fmt.Sprintf("%s_%s_%d", silly, thing, rng.Intn(9000)+1000))
		email := fmt.Sprintf("%s@parkfake.local", username)

		roll := rng.Intn(100)
		roleName := "petugas"
		if roll < 20 {
			roleName = "admin"
		} else if roll > 80 {
			roleName = "owner"
		}

		u := auth.User{
			ID:           uuid.New(),
			Name:         displayName,
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			RoleID:       roleIDs[roleName],
			IsActive:     rng.Intn(100) < 95,
			TenantID:     tenantID,
			CreatedAt:    time.Now().Add(-time.Duration(rng.Intn(24*30)) * time.Hour),
			UpdatedAt:    time.Now(),
		}
		db.Create(&u)
		out = append(out, u)
	}
	return out
}

func seedVehicleTypes(db *gorm.DB, rng *rand.Rand) []vehicle.VehicleType {
	types := []vehicle.VehicleType{
		{ID: uuid.New(), Name: "Motor", MinimumFee: 2000, Description: "Kendaraan roda 2"},
		{ID: uuid.New(), Name: "Mobil", MinimumFee: 5000, Description: "Kendaraan roda 4"},
		{ID: uuid.New(), Name: "Truk", MinimumFee: 10000, Description: "Kendaraan besar roda >4"},
	}
	for i := range types {
		types[i].CreatedAt = time.Now()
		db.Create(&types[i])
	}
	return types
}

func seedZonesAndGates(db *gorm.DB, rng *rand.Rand, tenantID uuid.UUID, createdBy *uuid.UUID) ([]zone.Zone, []zone.Gate, []zone.Gate) {
	zoneTemplates := []string{"Gedung Utama", "Basement A", "Lobby Belakang", "Area VIP"}
	var zones []zone.Zone
	var entryGates []zone.Gate
	var exitGates []zone.Gate

	for _, name := range zoneTemplates {
		z := zone.Zone{
			ID:          uuid.New(),
			Name:        name,
			Description: faker.Sentence(),
			Capacity:    rng.Intn(200) + 50,
			IsActive:    true,
			TenantID:    tenantID,
			CreatedBy:   createdBy,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		db.Create(&z)
		zones = append(zones, z)

		// Create Entry Gate
		entry := zone.Gate{
			ID:           uuid.New(),
			ZoneID:       z.ID,
			Name:         name + " Entry",
			GateType:     "entry",
			Mode:         "hybrid", // can be rfid or ticket
			LocationDesc: "Pintu Masuk " + name,
			GateToken:    zone.NewGateToken(),
			IsActive:     true,
			TenantID:     tenantID,
			CreatedBy:    createdBy,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&entry)
		entryGates = append(entryGates, entry)

		// Create Exit Gate
		exit := zone.Gate{
			ID:           uuid.New(),
			ZoneID:       z.ID,
			Name:         name + " Exit",
			GateType:     "exit",
			Mode:         "hybrid",
			LocationDesc: "Pintu Keluar " + name,
			GateToken:    zone.NewGateToken(),
			IsActive:     true,
			TenantID:     tenantID,
			CreatedBy:    createdBy,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&exit)
		exitGates = append(exitGates, exit)
	}
	return zones, entryGates, exitGates
}

func seedFeeConfigs(db *gorm.DB, rng *rand.Rand, zones []zone.Zone, vTypes []vehicle.VehicleType, tenantID uuid.UUID) []fee.FeeConfig {
	var cfgs []fee.FeeConfig
	for _, z := range zones {
		for _, vt := range vTypes {
			base := vt.MinimumFee
			cfg := fee.FeeConfig{
				ID:                 uuid.New(),
				ZoneID:             z.ID,
				VehicleTypeID:      vt.ID,
				BaseFee:            base,
				GracePeriodMinutes: 15,
				IsActive:           true,
				EffectiveFrom:      time.Now().AddDate(-1, 0, 0),
				TenantID:           tenantID,
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			db.Create(&cfg)
			cfgs = append(cfgs, cfg)

			// Simple tier structure
			// 1st hour: Base Fee
			// Next hours: BaseFee * 0.5 per hour (max 24h)
			tier1 := fee.FeeTier{
				ID:              uuid.New(),
				FeeConfigID:     cfg.ID,
				TierOrder:       1,
				DurationMinutes: 60,
				FeeAmount:       base,
				IsLastTier:      false,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			db.Create(&tier1)

			tier2 := fee.FeeTier{
				ID:              uuid.New(),
				FeeConfigID:     cfg.ID,
				TierOrder:       2,
				DurationMinutes: 60,
				FeeAmount:       base / 2,
				IsLastTier:      true, // means applies repetitively or to max duration limit
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			db.Create(&tier2)
		}
	}
	return cfgs
}

func seedVehicles(db *gorm.DB, rng *rand.Rand, vTypes []vehicle.VehicleType, tenantID uuid.UUID, count int) ([]vehicle.Vehicle, []vehicle.RFIDCard) {
	var vehicles []vehicle.Vehicle
	var rfidCards []vehicle.RFIDCard

	for i := 0; i < count; i++ {
		vt := vTypes[rng.Intn(len(vTypes))]
		vtID := vt.ID

		plateLetters := pick(rng, []string{"B ", "D ", "F ", "A ", "AE ", "Z "})
		plateNumbers := rng.Intn(9000) + 1000
		plateSuffix := pick(rng, []string{" XYZ", " ABC", " FCK", " GG", " BR"})
		plate := fmt.Sprintf("%s%d%s", plateLetters, plateNumbers, plateSuffix)

		v := vehicle.Vehicle{
			ID:            uuid.New(),
			PlateNumber:   plate,
			VehicleTypeID: &vtID,
			Source:        "manual",
			TenantID:      tenantID,
			CreatedAt:     time.Now().Add(-time.Duration(rng.Intn(24*90)) * time.Hour),
		}
		v.UpdatedAt = v.CreatedAt
		db.Create(&v)
		vehicles = append(vehicles, v)

		// 30% of vehicles get RFID cards
		if rng.Intn(100) < 30 {
			uid := fmt.Sprintf("%08X", rng.Uint32()) + fmt.Sprintf("%08X", rng.Uint32())
			card := vehicle.RFIDCard{
				ID:        uuid.New(),
				CardUID:   uid,
				VehicleID: v.ID,
				IsActive:  true,
				CreatedAt: v.CreatedAt,
			}
			db.Create(&card)
			rfidCards = append(rfidCards, card)
		}
	}
	return vehicles, rfidCards
}

func seedTransactions(
	db *gorm.DB,
	rng *rand.Rand,
	tenantID uuid.UUID,
	entryGates []zone.Gate,
	exitGates []zone.Gate,
	vehicles []vehicle.Vehicle,
	rfidCards []vehicle.RFIDCard,
	feeConfigs []fee.FeeConfig,
	users []auth.User,
	count int,
) {
	now := time.Now()

	for i := 0; i < count; i++ {
		// 1. Calculate Entry Time (Spread over last 30 days)
		daysAgo := rng.Intn(30)
		entryAt := now.Add(-time.Duration(daysAgo) * 24 * time.Hour).Add(-time.Duration(rng.Intn(24)) * time.Hour)

		// 2. Select Entry Gate
		egIdx := rng.Intn(len(entryGates))
		entryGate := entryGates[egIdx]
		zoneID := entryGate.ZoneID

		// 3. Status and Duration logic
		// 90% Completed, 5% Pending Payment, 5% Active
		statusRoll := rng.Intn(100)
		status := transaction.StatusCompleted
		isClosed := true
		if statusRoll < 5 {
			status = transaction.StatusPendingPayment
			isClosed = false
		} else if statusRoll < 10 {
			status = transaction.StatusActive
			isClosed = false
		}

		durationMins := rng.Intn(12*60) + 15 // Stay 15 mins to 12 hours
		exitAt := entryAt.Add(time.Duration(durationMins) * time.Minute)
		if exitAt.After(now) {
			exitAt = now
			durationMins = int(now.Sub(entryAt).Minutes())
		}

		// 4. Entry Method: RFID or Ticket
		var entryMethod string
		var rfidCardID *uuid.UUID
		var ticketCode *string
		var vehicleID *uuid.UUID
		var feeConfigID *uuid.UUID

		if len(rfidCards) > 0 && rng.Intn(100) < 30 {
			// RFID 
			entryMethod = transaction.MethodRFID
			card := rfidCards[rng.Intn(len(rfidCards))]
			rfidCardID = &card.ID
			vehicleID = &card.VehicleID
		} else {
			// Ticket
			entryMethod = transaction.MethodTicket
			code := fmt.Sprintf("TCK-%X", rng.Int31())
			ticketCode = &code
			// Usually we capture plate via OCR at entry, so tie a random vehicle
			v := vehicles[rng.Intn(len(vehicles))]
			vehicleID = &v.ID
		}

		// Find fee config for vehicle if we have vehicle
		if vehicleID != nil {
			var vtID *uuid.UUID
			for _, v := range vehicles {
				if v.ID == *vehicleID {
					vtID = v.VehicleTypeID
					break
				}
			}
			if vtID != nil {
				for _, fc := range feeConfigs {
					if fc.ZoneID == zoneID && fc.VehicleTypeID == *vtID {
						feeConfigID = &fc.ID
						break
					}
				}
			}
		}

		// 5. Build Transaction
		tx := transaction.Transaction{
			ID:              uuid.New(),
			TransactionCode: fmt.Sprintf("TRX-%s-%X", entryAt.Format("060102"), rng.Int31()),
			EntryGateID:     entryGate.ID,
			EntryMethod:     entryMethod,
			RFIDCardID:      rfidCardID,
			TicketCode:      ticketCode,
			EntryAt:         entryAt,
			EntryPhotoURL:   "https://eu2.contabostorage.com/fake-bucket/entry-photo.jpg",
			VehicleID:       vehicleID,
			FeeConfigID:     feeConfigID,
			Status:          status,
			ZoneID:          zoneID,
			TenantID:        tenantID,
			CreatedAt:       entryAt,
			UpdatedAt:       entryAt,
		}

		totalFee := 0
		if feeConfigID != nil && isClosed {
			// Calculate dummy fee
			var cfgBase int
			for _, fc := range feeConfigs {
				if fc.ID == *feeConfigID {
					cfgBase = fc.BaseFee
					break
				}
			}
			// Simulate calculation
			// first hour = base, next hours = base/2
			hours := durationMins / 60
			if durationMins%60 > 0 {
				hours++
			}
			if hours > 0 {
				totalFee = cfgBase + ((hours - 1) * (cfgBase / 2))
			}
		}

		tx.CalculatedFee = totalFee

		var assignedExitGate *zone.Gate
		if isClosed || status == transaction.StatusPendingPayment {
			// Find exit gate in same zone
			var matchingExits []zone.Gate
			for _, eg := range exitGates {
				if eg.ZoneID == zoneID {
					matchingExits = append(matchingExits, eg)
				}
			}
			if len(matchingExits) > 0 {
				assignedExitGate = &matchingExits[rng.Intn(len(matchingExits))]
				tx.ExitGateID = &assignedExitGate.ID
			}

			// Complete transaction details
			em := tx.EntryMethod
			tx.ExitMethod = &em
			tx.ExitAt = &exitAt
			tx.ExitPhotoURL = "https://eu2.contabostorage.com/fake-bucket/exit-photo.jpg"
			tx.UpdatedAt = exitAt
		}

		db.Create(&tx)

		// 6. Transaction Logs
		sendLog(db, tx.ID, "", transaction.StatusActive, "scan_entry", "system", nil, tx.CreatedAt)
		if isClosed || status == transaction.StatusPendingPayment {
			sendLog(db, tx.ID, transaction.StatusActive, transaction.StatusPendingPayment, "exit_scanned", "system", nil, exitAt)
		}
		if isClosed {
			operator := users[rng.Intn(len(users))]
			sendLog(db, tx.ID, transaction.StatusPendingPayment, transaction.StatusCompleted, "payment_success", "user", &operator.ID, exitAt.Add(1*time.Minute))
			
			// 7. Seed Payment
			method := payment.MethodCash
			if rng.Intn(100) > 70 {
				method = payment.MethodQRIS
			}
			
			py := payment.Payment{
				ID:              uuid.New(),
				TransactionID:   tx.ID,
				Method:          method,
				Amount:          totalFee,
				Status:          payment.StatusCompleted,
				HandledByUserID: &operator.ID,
				TenantID:        tenantID,
				CreatedAt:       exitAt,
				UpdatedAt:       exitAt.Add(1 * time.Minute),
			}
			
			if method == payment.MethodCash {
				py.CashTendered = ((totalFee / 10000) + 1) * 10000
				py.CashChange = py.CashTendered - totalFee
				py.MidtransOrderID = fmt.Sprintf("CASH-%s", tx.ID.String()[:13])
			} else {
				py.MidtransOrderID = fmt.Sprintf("ORDER-%s", tx.ID.String()[:8])
			}
			
			t := exitAt.Add(1 * time.Minute)
			py.PaidAt = &t
			db.Create(&py)
		}
	}
}

func sendLog(db *gorm.DB, txID uuid.UUID, fromStatus, toStatus, event, triggeredBy string, triggeredByUser *uuid.UUID, ts time.Time) {
	lg := transaction.TransactionLog{
		ID:                uuid.New(),
		TransactionID:     txID,
		FromStatus:        fromStatus,
		ToStatus:          toStatus,
		Event:             event,
		TriggeredBy:       triggeredBy,
		TriggeredByUserID: triggeredByUser,
		CreatedAt:         ts,
	}
	db.Create(&lg)
}


// -------------------------------------------------------------
// HELPERS
// -------------------------------------------------------------

func loadRoleIDs(db *gorm.DB) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	for _, name := range []string{"admin", "petugas", "owner"} {
		var r auth.Role
		if err := db.Where("name = ?", name).First(&r).Error; err != nil {
			return nil, fmt.Errorf("role %q: %w", name, err)
		}
		out[name] = r.ID
	}
	return out, nil
}

func pick(rng *rand.Rand, list []string) string {
	return list[rng.Intn(len(list))]
}

func mustUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		log.Fatalf("invalid UUID %q: %v", s, err)
	}
	return id
}
