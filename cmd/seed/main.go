// Command seed inserts demo categories/products/variants for local
// development. Safe to re-run — it upserts by slug/sku.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"github.com/esrid/maison/internal/auth"
	"github.com/esrid/maison/internal/config"
	"github.com/esrid/maison/internal/db"
)

// Dev-only bootstrap credentials — override via env for anything beyond a
// local sandbox. There's no admin CLI yet; this is the only way to get a
// first admin account.
const (
	devAdminEmail           = "admin@example.com"
	devAdminPasswordEnvVar  = "SEED_ADMIN_PASSWORD"
	devAdminPasswordDefault = "changeme123"
)

type seedVariant struct {
	sku, name  string
	options    map[string]string
	priceCents int64
	stockQty   int
}

type seedProduct struct {
	slug, title, description, category string
	variants                           []seedVariant
	images                             []string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ctx := context.Background()
	if err := seed(ctx, conn); err != nil {
		log.Fatal(err)
	}
	if err := seedAdmin(ctx, conn); err != nil {
		log.Fatal(err)
	}
	log.Println("seed complete")
}

func seedAdmin(ctx context.Context, conn *sql.DB) error {
	password := os.Getenv(devAdminPasswordEnvVar)
	if password == "" {
		password = devAdminPasswordDefault
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `
		INSERT INTO admin_users (email, password_hash) VALUES (?, ?)
		ON CONFLICT(email) DO UPDATE SET password_hash = excluded.password_hash`,
		devAdminEmail, hash)
	if err != nil {
		return err
	}
	log.Printf("admin user ready: %s / (password from %s, default %q)", devAdminEmail, devAdminPasswordEnvVar, devAdminPasswordDefault)
	return nil
}

func seed(ctx context.Context, conn *sql.DB) error {
	categories := map[string]int64{}
	for _, c := range []struct{ slug, name string }{
		{"apparel", "Vêtements"},
		{"home", "Maison"},
		{"accessories", "Accessoires"},
	} {
		res, err := conn.ExecContext(ctx,
			`INSERT INTO categories (slug, name) VALUES (?, ?) ON CONFLICT(slug) DO UPDATE SET name = excluded.name`,
			c.slug, c.name)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if id == 0 {
			if err := conn.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = ?`, c.slug).Scan(&id); err != nil {
				return err
			}
		}
		categories[c.slug] = id
	}

	products := []seedProduct{
		{
			slug: "linen-shirt", title: "Chemise en Lin", category: "apparel",
			description: "Coupe décontractée, lin lavé pour une douceur incomparable. Taille normale.",
			images:      []string{"/static/img/product-2.jpg"},
			variants: []seedVariant{
				{"SHIRT-S", "Small", map[string]string{"size": "S"}, 6800, 8},
				{"SHIRT-M", "Medium", map[string]string{"size": "M"}, 6800, 3},
				{"SHIRT-L", "Large", map[string]string{"size": "L"}, 6800, 0},
			},
		},
		{
			slug: "canvas-tote", title: "Sac Cabas", category: "accessories",
			description: "Cabas en coton épais à bandoulières renforcées. Pour votre quotidien.",
			images:      []string{"/static/img/product-1.jpg"},
			variants: []seedVariant{
				{"TOTE-NAT", "Naturel", map[string]string{"color": "Naturel"}, 3200, 40},
				{"TOTE-BLK", "Noir", map[string]string{"color": "Noir"}, 3200, 12},
			},
		},
		{
			slug: "ceramic-mug", title: "Tasse Céramique", category: "home",
			description: "Tasse en grès émaillé à la main, 35cl. Passe au micro-ondes et au lave-vaisselle.",
			images:      []string{"/static/img/product-3.jpg"},
			variants: []seedVariant{
				{"MUG-STD", "Standard", map[string]string{}, 1800, 60},
			},
		},
	}

	for _, p := range products {
		catID := categories[p.category]
		var productID int64
		err := conn.QueryRowContext(ctx, `
			INSERT INTO products (category_id, slug, title, description, status)
			VALUES (?, ?, ?, ?, 'published')
			ON CONFLICT(slug) DO UPDATE SET title = excluded.title, description = excluded.description, status = 'published'
			RETURNING id`,
			catID, p.slug, p.title, p.description).Scan(&productID)
		if err != nil {
			return err
		}

		for i, url := range p.images {
			if _, err := conn.ExecContext(ctx, `
				INSERT INTO product_images (product_id, url, alt, position) VALUES (?, ?, ?, ?)
				ON CONFLICT DO NOTHING`, productID, url, p.title, i); err != nil {
				return err
			}
		}

		for _, v := range p.variants {
			optsJSON, err := json.Marshal(v.options)
			if err != nil {
				return err
			}
			if _, err := conn.ExecContext(ctx, `
				INSERT INTO product_variants (product_id, sku, name, options_json, price_cents, stock_qty, low_stock_threshold)
				VALUES (?, ?, ?, ?, ?, ?, 5)
				ON CONFLICT(sku) DO UPDATE SET price_cents = excluded.price_cents, stock_qty = excluded.stock_qty`,
				productID, v.sku, v.name, string(optsJSON), v.priceCents, v.stockQty); err != nil {
				return err
			}
		}
	}

	return nil
}
