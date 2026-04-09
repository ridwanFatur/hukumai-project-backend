package db

import (
	"encoding/json"

	"github.com/ridwanFatur/hukumai-project-backend/models"
)

func SeedSubscriptionPlans() {
	type planSeed struct {
		Name      string
		Slug      string
		Price     int64
		Features  []string
		SortOrder int
	}

	seeds := []planSeed{
		{
			Name:  "Free",
			Slug:  "free",
			Price: 0,
			Features: []string{
				"5 pertanyaan chatbot / hari",
				"Pencarian regulasi dasar",
				"Ringkasan 3 dokumen / bulan",
				"Akses komunitas",
			},
			SortOrder: 1,
		},
		{
			Name:  "Pro",
			Slug:  "pro",
			Price: 299000,
			Features: []string{
				"Pertanyaan chatbot tak terbatas",
				"Pencarian & filter lanjutan",
				"Ringkasan dokumen tak terbatas",
				"Analisis risiko kontrak",
				"Draft dokumen otomatis",
				"Notifikasi regulasi real-time",
			},
			SortOrder: 2,
		},
		{
			Name:  "Enterprise",
			Slug:  "enterprise",
			Price: 0,
			Features: []string{
				"Semua fitur Pro",
				"Multi-pengguna & tim",
				"Akses API penuh",
				"Dashboard analitik lanjutan",
				"Integrasi kustom",
				"Dukungan prioritas 24/7",
			},
			SortOrder: 3,
		},
	}

	for _, s := range seeds {
		var existing models.SubscriptionPlan
		if DB.Where("slug = ?", s.Slug).First(&existing).Error == nil {
			continue
		}
		features, _ := json.Marshal(s.Features)
		plan := models.SubscriptionPlan{
			Name:      s.Name,
			Slug:      s.Slug,
			Price:     s.Price,
			Currency:  "IDR",
			Interval:  "month",
			Features:  string(features),
			IsActive:  true,
			SortOrder: s.SortOrder,
		}
		DB.Create(&plan)
	}
}
