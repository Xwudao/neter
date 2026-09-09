package gen

import (
	"go/format"
	"strings"
	"testing"
	"text/template"

	"github.com/Xwudao/neter/internal/tpl"
)

func TestSQLCTemplatesRender(t *testing.T) {
	g := &Generator{
		Name:          "product",
		PackageName:   "biz",
		ModName:       "example.com/app",
		Model:         "Product",
		Plural:        "Products",
		Table:         "products",
		IsSQLC:        true,
		WithCRUD:      true,
		StructBizName: "ProductBiz",
	}

	cases := []struct {
		name     string
		tplText  string
		contains []string
	}{
		{
			name:    "biz",
			tplText: tpl.BizSQLCTpl,
			contains: []string{
				"type ProductRepository interface {",
				"GetAll(ctx context.Context) ([]*sqlc.Product, error)",
				"Create(ctx context.Context) (*sqlc.Product, error)",
			},
		},
		{
			name:    "repo",
			tplText: tpl.RepoSQLCTpl,
			contains: []string{
				"var _ biz.ProductRepository = (*productRepository)(nil)",
				"db/query/products.sql",
				"ListProducts",
				"func (r *productRepository) GetByID(ctx context.Context, id int64) (*sqlc.Product, error)",
			},
		},
		{
			name:    "biz_iface",
			tplText: tpl.BizIfaceSQLCTpl,
			contains: []string{
				"type ProductBizIface interface {",
				"var _ ProductBizIface = (*ProductBiz)(nil)",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := template.New(tc.name).Parse(tc.tplText)
			if err != nil {
				t.Fatalf("parse template: %v", err)
			}
			var b strings.Builder
			if err := parsed.Execute(&b, g); err != nil {
				t.Fatalf("execute template: %v", err)
			}
			if _, err := format.Source([]byte(b.String())); err != nil {
				t.Fatalf("rendered template is not valid Go: %v\n%s", err, b.String())
			}
			for _, want := range tc.contains {
				if !strings.Contains(b.String(), want) {
					t.Errorf("rendered %s missing %q\n%s", tc.name, want, b.String())
				}
			}
			if strings.Contains(b.String(), "internal/data/ent") {
				t.Errorf("sqlc template %s must not reference ent\n%s", tc.name, b.String())
			}
		})
	}
}

func TestLegacyTemplatesStillReferenceEnt(t *testing.T) {
	g := &Generator{
		Name:          "thing",
		PackageName:   "biz",
		ModName:       "example.com/app",
		EntName:       "Thing",
		WithCRUD:      true,
		StructBizName: "ThingBiz",
	}
	parsed, err := template.New("biz").Parse(tpl.BizTpl)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := parsed.Execute(&b, g); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "ent.Thing") {
		t.Fatalf("legacy biz template should reference ent.Thing\n%s", b.String())
	}
}
