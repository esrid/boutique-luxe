## 2026-07-08 DESIGN BRIEF: Maison — Améliorations UI/UX E-commerce

### Recherche effectuée
Étude des patterns e-commerce luxe : Zara, Mango, COS, SSENSE, SKIMS, Aesop.

### Priorités (10 patterns, ordre d'impact)

| # | Pattern | Effort | Impact |
|---|---------|--------|--------|
| 1 | **Mega-menu hover** (images catégories) | CSS | 🟢 Énorme |
| 2 | **Quick add to cart** (hover carte produit) | CSS+JS | 🟢 Énorme |
| 3 | **Sticky add-to-cart** (mobile) | CSS | 🟡 Important |
| 4 | **Breadcrumbs** (pages produit) | HTML | 🟢 Rapide |
| 5 | **Free shipping bar** (panier) | HTML+CSS | 🟢 Rapide |
| 6 | **Mobile hamburger** (menu slide-out) | CSS+JS | 🟡 Important |
| 7 | **Image zoom** (hover carte produit) | CSS | 🟢 Rapide |
| 8 | **Quantity +/- buttons** (panier) | HTML+CSS | 🟢 Rapide |
| 9 | **Loading skeleton** (suspense) | CSS | 🟡 Important |
| 10 | **Quick view modal** | CSS+JS | 🔴 Complexe |

### Design System — inchangé (Zara minimaliste)
- Monochrome noir/blanc
- Inter sans-serif
- Bordures droites
- Photos full-bleed

### Go Template Contracts
- `/` — home (existant)
- `/products` — catalogue (existant)
- `/products/{slug}` — fiche produit (existant)
- `/cart` — panier (existant)
- `/checkout` — commande (existant)
- Les routes ne changent PAS, on améliore le CSS + HTML existant

### What Frontend Should Build
- [ ] Mega-menu avec images par catégorie (hover)
- [ ] Quick add to cart au hover des cartes produit
- [ ] Sticky add-to-cart sur mobile
- [ ] Breadcrumbs sur `/products/{slug}`
- [ ] Free shipping progress bar dans `/cart`
- [ ] Mobile hamburger menu
- [ ] Image zoom subtil au hover (scale 1.03, pas plus)
- [ ] Quantity +/- buttons dans le panier

### Don't Do
- ❌ Pas changer la palette (rester monochrome Zara)
- ❌ Pas de popups agressives
- ❌ Pas de carrousel hero
- ❌ Pas de "dark mode" dashboard

**STATUS: Brief complete — frontend-engineer can start building.**
