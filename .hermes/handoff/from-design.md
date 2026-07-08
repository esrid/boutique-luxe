# Design Brief — MAISON

> **Design brief transmis au frontend-engineer**  
> Template e-commerce Go (html/template SSR) → relooké boutique luxe éditoriale.
> Tout le CSS est moderne (design tokens sur `:root`, pas de préprocesseur).
> Responsive intrinsic (grid + auto-fit / auto-fill). Aucun media query dans les composants.
> Aucun client-side rendering. Aucune dépendance JS pour le layout.

---

## 1. Identité de marque

| Élément | Valeur |
|---|---|
| **Nom** | Maison |
| **URL** | (à définir, dispo via config) |
| **Tagline** | *« L'élégance silencieuse. »* |
| **Positionnement** | Luxe éditorial sobre — pas de bling, pas de slogans agressifs. L'expérience d'achat doit évoquer une galerie ou un magazine de mode haut de gamme. |
| **Cible** | Femme & homme, 28–55 ans, urbain, cultivé, sensible au design et aux matières. |
| **Valeurs** | Slow fashion, intemporalité, artisanat, matériaux nobles, garde-robe durable. |
| **Ton** | Poétique mais précis. Court. Pas de « stuff » marketing. Descriptifs sensoriels (toucher, tombé, poids du tissu). |

**Phrases qui incarnent la marque :**
- « Des lignes pures, des matières qui durent. »
- « Chaque pièce a une histoire. »
- « Moins, mais mieux. »

---

## 2. Mood & Inspiration

### Ambiance générale
- Épuré, calme, spacieux — beaucoup d'espace négatif
- Lumière naturelle douce (aube / fin d'après-midi)
- Matières brutes : lin, pierre calcaire, bois clair, papier texturé
- Photographie : fond blanc cassé chaud, modèle en mouvement, grain argentique léger
- Pas d'animations flashy ; transitions en douceur (opacité, transform subtil)

### Sites de référence
| Source | Ce qu'on en prend |
|---|---|
| **SSENSE** | Grille asymétrique, typo forte, hero éditorial à défilement lent |
| **Totême** | Palette neutre, mannequin en situation, fiche produit minimaliste |
| **The Row** | Luxury du silence, pas de logo ostentatoire, blanc cassé, mise en page éditoriale |
| **Acne Studios** | Cropping audacieux, typo serif, inventivité dans les grids |
| **Khaite** | Noir et blanc chaud, textures, atmosphère cinématographique |

### Mots-clés visuels
`luxe silencieux` `éditorial` `minéral` `matières brutes` `grain` `ombres portées` `monochrome chaud` `or discret`

---

## 3. Palette — Design Tokens OKLCH

> Aucun `#fff` ni `#000` nulle part. Les valeurs absolues sont remplacées par ces tokens.
> Tous les tokens sont déclarés sur `:root` dans `tokens.css`.

### 3.1 Tokens de couleur

```css
:root {
  /* ── Fond ────────────────────────────────── */
  --color-bg:         oklch(0.97 0.008 90);    /* Blanc cassé chaud — fond principal */
  --color-bg-alt:     oklch(0.94 0.012 85);    /* Beige très clair — fond secondaire (hero, cartes) */
  --color-bg-elevated: oklch(0.92 0.015 82);   /* Beige doux — hover, surfaces élevées */

  /* ── Texte ───────────────────────────────── */
  --color-text:       oklch(0.13 0.02 270);    /* Charbon profond (presque noir mais chaud) */
  --color-text-soft:  oklch(0.35 0.025 260);   /* Gris foncé chaud — meta, labels secondaires */
  --color-text-muted: oklch(0.55 0.025 250);   /* Gris moyen — placeholder, discret */

  /* ── Accent Or / Champagne ───────────────── */
  --color-accent:         oklch(0.72 0.11 68); /* Or champagne — boutons, liens, accents */
  --color-accent-hover:   oklch(0.62 0.12 68); /* Or plus profond — hover */
  --color-accent-light:   oklch(0.82 0.06 70); /* Or clair — badges, tags */
  --color-accent-bg:      oklch(0.92 0.03 72); /* Fond teinté or très léger */

  /* ── Surfaces / Conteneurs ───────────────── */
  --color-surface:      oklch(0.96 0.006 85);  /* Fond de carte, modale */
  --color-surface-hover: oklch(0.93 0.01 83);  /* Hover sur carte */
  --color-border:       oklch(0.85 0.012 80);  /* Bordures fines */
  --color-border-light: oklch(0.90 0.008 82);  /* Bordures très légères */

  /* ── États ───────────────────────────────── */
  --color-error:    oklch(0.55 0.15 25);  /* Rouge profond terreux */
  --color-success:  oklch(0.60 0.10 145); /* Vert sauge discret */
  --color-info:     oklch(0.60 0.08 240); /* Bleu pâle minéral */

  /* ── Overlay ──────────────────────────────── */
  --color-overlay: oklch(0 0 0 / 0.35);  /* Voile sombre pour modales */
}
```

### 3.2 Tokens d'espacement & rayon

```css
:root {
  --space-xs:  0.25rem;
  --space-sm:  0.5rem;
  --space-md:  1rem;
  --space-lg:  1.5rem;
  --space-xl:  2.5rem;
  --space-2xl: 4rem;
  --space-3xl: 6rem;

  --radius-sm: 2px;
  --radius-md: 4px;
  --radius-lg: 8px;
  --radius-full: 9999px;
}
```

### 3.3 Tokens d'ombre

```css
:root {
  --shadow-sm: 0 1px 2px oklch(0 0 0 / 0.04);
  --shadow-md: 0 2px 8px oklch(0 0 0 / 0.06);
  --shadow-lg: 0 4px 24px oklch(0 0 0 / 0.08);
  --shadow-xl: 0 8px 48px oklch(0 0 0 / 0.10);
}
```

### 3.4 Règle absolue

> **Aucun `#fff`, `#000`, `white`, `black`, `rgb(255 255 255)`, `rgb(0 0 0)` n'apparaît dans le CSS.**  
> Toute surface claire → `var(--color-bg)`. Tout texte foncé → `var(--color-text)`.  
> Les ombres utilisent `oklch(0 0 0 / N)` — jamais `rgba(0,0,0,N)`.

---

## 4. Typographie

### 4.1 Hiérarchie

| Usage | Police | Poids | Taille (fluid clamp) | Tracking |
|---|---|---|---|---|
| **Logo / Marque** | Playfair Display | 700 italic | `clamp(1.5rem, 2.5vw, 2.25rem)` | `0.02em` |
| **Titre H1** (hero) | Playfair Display | 400 | `clamp(2.5rem, 5vw, 4.5rem)` | `-0.02em` |
| **Titre H2** (section) | Playfair Display | 400 italic | `clamp(1.75rem, 3vw, 2.75rem)` | `0` |
| **Titre H3** (carte) | Playfair Display | 400 | `clamp(1.125rem, 1.8vw, 1.5rem)` | `0.01em` |
| **Titre H4** (meta) | Inter | 500 | `clamp(0.875rem, 1.2vw, 1rem)` | `0.04em` |
| **Corps** | Inter | 300 / 400 | `clamp(0.9375rem, 1.2vw, 1.125rem)` | `0.01em` |
| **Corps petit** (prix, filtres) | Inter | 400 | `clamp(0.8125rem, 1vw, 0.9375rem)` | `0.02em` |
| **Caption** (badges, tags) | Inter | 500 | `clamp(0.6875rem, 0.9vw, 0.8125rem)` | `0.06em` |
| **Navigation** | Inter | 400 | `clamp(0.8125rem, 1.2vw, 1rem)` | `0.05em` |
| **Bouton** | Inter | 500 | `clamp(0.8125rem, 1.2vw, 1rem)` | `0.06em` |

### 4.2 Déclarations CSS

```css
:root {
  --font-display: 'Playfair Display', 'Georgia', 'Times New Roman', serif;
  --font-body:    'Inter', -apple-system, 'Segoe UI', Roboto, sans-serif;

  /* Échelle typographique fluide */
  --text-caption:  clamp(0.6875rem, 0.9vw, 0.8125rem);
  --text-small:    clamp(0.8125rem, 1vw, 0.9375rem);
  --text-body:     clamp(0.9375rem, 1.2vw, 1.125rem);
  --text-nav:      clamp(0.8125rem, 1.2vw, 1rem);
  --text-btn:      clamp(0.8125rem, 1.2vw, 1rem);
  --text-h4:       clamp(0.875rem, 1.2vw, 1rem);
  --text-h3:       clamp(1.125rem, 1.8vw, 1.5rem);
  --text-h2:       clamp(1.75rem, 3vw, 2.75rem);
  --text-h1:       clamp(2.5rem, 5vw, 4.5rem);

  /* Hauteur de ligne */
  --leading-tight:   1.1;
  --leading-normal:  1.5;
  --leading-relaxed: 1.7;

  /* Épaisseurs */
  --weight-light:  300;
  --weight-regular: 400;
  --weight-medium: 500;
  --weight-bold:   700;
}
```

### 4.3 Intégration Google Fonts

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500&family=Playfair+Display:ital@0;1&display=swap" rel="stylesheet">
```

> ⚠️ Si les Google Fonts sont bloquées en production (RGPD / self-hosted), prévoir le fallback vers Georgia (serif) et system-ui (sans-serif). Les tailles `clamp()` restent inchangées.

---

## 5. Composants

Chaque composant ci-dessous a un nom de classe BEM (`maison__<composant>`) ou un nom atomique selon la convention du projet. L'important est la sémantique visuelle — le nommage s'adapte au pattern existant.

### 5.1 Navbar (`maison-navbar`)

```
┌──────────────────────────────────────────────────────┐
│  MAISON              NOUVEAUTÉS  FEMME  HOMME  [🔍] │
│                         COLLECTIONS         [🛒 0]  │
├──────────────────────────────────────────────────────┤
│                (ligne fine — var(--color-border))     │
└──────────────────────────────────────────────────────┘
```

- **Logo** à gauche, Playfair Display italic 700
- **Liens** centrés ou légèrement décalés (Inter, tracking 0.05em, uppercase)
- **Icônes** (recherche, panier, compte) à droite
- **Sticky** au scroll avec `backdrop-filter: blur(8px)` et fond `var(--color-bg) / 0.85`
- **Barre fine** en bas (`1px solid var(--color-border-light)`)
- **Aucun hamburger menu** — pas de responsive collapse. Les liens passent en plus petit ou en colonne sur mobile large, mais restent visibles.

### 5.2 Hero (`maison-hero`)

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│                                                          │
│   ┌──────────────────┐                                   │
│   │  Image éditoriale │                                   │
│   │  (grand écran)    │   NOM DE LA COLLECTION           │
│   │                   │   Sous-titre court               │
│   │                   │   [Découvrir]                     │
│   └──────────────────┘                                   │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

- **Layout** : image à gauche ou en fond, texte superposé ou à droite selon le template
- **Image** ratio 4:5 ou 16:9 (selon page), object-fit: cover
- **Typo** H1 sur l'image avec voile `var(--color-overlay)`
- **Bouton** "Découvrir" — filled accent, avec hover qui éclaircit subtilement
- **Pas de slider automatique** — un hero statique, ou navigation manuelle (flèches)

### 5.3 Product Card (`maison-product-card`)

```
┌───────────────┐
│               │
│   ┌───────┐   │
│   │ Image │   │   Ratio 3:4 (portrait)
│   │ 3:4   │   │   Object-fit: cover
│   │       │   │   Hover: scale(1.02) doux + ombre
│   └───────┘   │
│               │
│  Nom du produit
│  € 290,00
│  Couleurs : ● ○ ●
└───────────────┘
```

- **Image** : ratio 3:4 strict (`aspect-ratio: 3/4`)
- **Hover** : `transform: scale(1.02)` + `box-shadow` augmenté. Transition 400ms ease.
- **Texte sous l'image** : nom en Playfair H3, prix en Inter small
- **Variantes** : pastilles de couleur (rondes, 16px, border fine) — seulement les couleurs disponibles
- **Badge** optionnel "Nouveau" / "Épuisé" en accent-light

### 5.4 Boutons

```css
/* Filled — pour CTA primaires */
.maison-btn--filled {
  background: var(--color-accent);
  color: var(--color-bg);
  border: none;
  padding: var(--space-sm) var(--space-xl);
  font: var(--weight-medium) var(--text-btn) var(--font-body);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  transition: background 300ms ease;
}
.maison-btn--filled:hover {
  background: var(--color-accent-hover);
}

/* Outline — pour actions secondaires */
.maison-btn--outline {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-text);
  padding: var(--space-sm) var(--space-xl);
  font: var(--weight-medium) var(--text-btn) var(--font-body);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  transition: all 300ms ease;
}
.maison-btn--outline:hover {
  background: var(--color-text);
  color: var(--color-bg);
}

/* Ghost — pour actions tertiaires (liens, filtres) */
.maison-btn--ghost {
  background: transparent;
  color: var(--color-text-soft);
  border: none;
  padding: var(--space-xs) var(--space-sm);
  font: var(--weight-regular) var(--text-small) var(--font-body);
  letter-spacing: 0.04em;
}
.maison-btn--ghost:hover {
  color: var(--color-accent);
}
```

### 5.5 Filters (page Collection)

```
┌──────────────────────────────────────────────────┐
│ [Toutes]  [Nouveautés]  [Vêtements]  [Accessoires] │  ← catégories (ghost btns)
│                                                    │
│ Catégorie    Taille    Couleur    Prix    Trier par │  ← accordéon horizontal
│ ──────────   ──────    ───────    ────    ──────── │
└──────────────────────────────────────────────────┘
```

- Sur desktop : rangée horizontale de dropdowns accordéon
- Sur tablette/mobile : ligne empilée (wrap) — pas de modale, pas de drawer
- Chaque filtre s'ouvre au clic, se ferme au clic ailleurs
- **Aucun JS requis pour le layout** — l'accordéon est progressive enhancement

### 5.6 Variant List (page Produit)

```
Couleur :  ● Noir  ● Beige  ● Sable  ─── pastilles cliquables
Taille :   [36] [38] [40] [42] [44]
           (tailles dispo en filled, épuisées barrées / grisées)
```

- Pastilles couleur : 24px, cercle, `outline: 1px solid var(--color-border)`, sélection = bordure accent
- Tailles : boutons outline, sélection = filled text
- État épuisé : `opacity: 0.35` + `text-decoration: line-through`

### 5.7 Cart Item (page Panier)

```
┌──────────────────────────────────────────────────────────┐
│ ┌──────────┐                                             │
│ │ Image    │  Nom du produit                       € 290 │
│ │ 80×100   │  Couleur : Sable  Taille : 38               │
│ └──────────┘  [−]  1  [+]                          ✕     │
│                                                          │
├──────────────────────────────────────────────────────────┤
```

- Image thumbnail 80×100 (respecte ratio 4:5 crop)
- Quantité : boutons − / + avec input au centre, design minimal
- Supprimer : icône ✕ ou texte "Supprimer" en ghost btn

### 5.8 Footer (`maison-footer`)

```
┌──────────────────────────────────────────────────────────┐
│  MAISON                                                    │
│  (tagline)                                                 │
│                                                            │
│  Boutique    Service      Informations    Suivez-nous     │
│  Nouveautés  Livraison    À propos        Instagram       │
│  Femme       Retours      Notre histoire  Pinterest       │
│  Homme       FAQ          Engagements      Newsletter      │
│  Collections Contact      Carrières       [email input]    │
│                                                            │
│  ────────────────────────────────────────────────────────  │
│  © 2025 Maison. Tous droits réservés.   CGV  Mentions     │
└──────────────────────────────────────────────────────────┘
```

- Fond `var(--color-bg-alt)`
- 4 colonnes sur desktop, 2 sur tablette, 1 sur mobile
- Newsletter : input ghost + bouton accent inline
- Pas de gros blocs — aération, espacement généreux

---

## 6. Pages

### 6.1 Home (`home.html`)

```
┌─ HERO ───────────────────────────────────────────────┐
│  Image grand format (éditorial)                       │
│  MAISON                                               │
│  « L'élégance silencieuse. »                          │
│  [Découvrir la collection]                            │
├─ COLLECTIONS ────────────────────────────────────────┤
│  [FEMME]   [HOMME]   [ACCESSOIRES]                   │
│  3 cartes éditoriales (image + nom), lien vers        │
│  chaque collection                                    │
├─ VALEURS ────────────────────────────────────────────┤
│  3 colonnes : Artisanat / Matières nobles /           │
│  Intemporalité (icône + titre + phrase)               │
├─ NEWSLETTER ─────────────────────────────────────────┤
│  Texte court + input email + bouton                  │
├─ FOOTER ─────────────────────────────────────────────┤
└──────────────────────────────────────────────────────┘
```

**Sections :**
1. **Hero** — image pleine largeur (ou presque), titre éditorial, bouton CTA
2. **Collections** — 3 cartes cliquables en grille (`auto-fit`, min 280px)
3. **Nos valeurs** — 3 blocs avec icône (SVG inline ou Unicode sobre), titre H3, texte body
4. **Newsletter** — fond alterné `var(--color-bg-alt)`, centré, minimal
5. **Footer**

### 6.2 Collection / Products (`products.html`)

```
┌─ BREADCRUMB ────────────────────────────────────────┐
│  Accueil  /  Femme  /  Robes                        │
├─ HEADER ────────────────────────────────────────────┤
│  Robes (H2)                                         │
│  24 pièces — Description éditoriale courte          │
├─ FILTRES ───────────────────────────────────────────┤
│  [Catégorie] [Taille] [Couleur] [Prix] [Trier par] │
├─ GRILLE PRODUITS ───────────────────────────────────┤
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐               │
│  │ Card │ │ Card │ │ Card │ │ Card │               │
│  └──────┘ └──────┘ └──────┘ └──────┘               │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐               │
│  │ Card │ │ Card │ │ Card │ │ Card │               │
│  └──────┘ └──────┘ └──────┘ └──────┘               │
├─ PAGINATION ────────────────────────────────────────┤
│  ‹  1  2  3 …  12  ›                               │
├─ FOOTER ────────────────────────────────────────────┤
└──────────────────────────────────────────────────────┘
```

- Grille : `display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr))`
- Pas de media query pour changer le nombre de colonnes
- Pagination : boutons numérotés outline, page active = filled

### 6.3 Product Detail (`product.html`)

```
┌──────────────────────────────────────────────────────────┐
│  Breadcrumb : Accueil / Femme / Robes / Robe Soline      │
├──────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────────────────────────────┐  │
│  │            │  │  Nom du produit (H2)                │  │
│  │  Image     │  │  € 290,00                           │  │
│  │  produit   │  │  ──────────────────────────         │  │
│  │  (3:4)     │  │  Description éditoriale courte.     │  │
│  │            │  │  Matière : 100% soie sauvage.       │  │
│  │            │  │  Coupe : droite, longueur genou.    │  │
│  │            │  │                                     │  │
│  │  ┌──┐┌──┐  │  │  Couleur : ● Noir  ● Beige  ● Sable│  │
│  │  │  ││  │  │  │  Taille  : [36] [38] [40] [42] [44]│  │
│  │  └──┘└──┘  │  │                                     │  │
│  │  (miniatures)│  │  [Ajouter au panier]   [★ Favoris]│  │
│  │            │  │                                     │  │
│  └────────────┘  │  ──────────────────────────         │  │
│                   │  Livraison gratuite dès 300€        │  │
│                   │  Retours sous 30 jours              │  │
│                   │  ──────────────────────────         │  │
│                   │  Entretien : lavage à froid, séchage│  │
│                   │  à l'air libre.                     │  │
│                   └────────────────────────────────────┘  │
├─ RECOMMANDATIONS ───────────────────────────────────────┤
│  Vous aimerez aussi : [Card] [Card] [Card] [Card]       │
├─ FOOTER ────────────────────────────────────────────────┤
└──────────────────────────────────────────────────────────┘
```

- Layout 2 colonnes : image (gauche) + infos (droite)
- Sticky sur le panneau d'infos si assez de hauteur
- Miniatures en dessous de l'image principale
- Bouton "Ajouter au panier" filled accent, large

### 6.4 Cart (`cart.html`)

```
┌──────────────────────────────────────────────────────────┐
│  Mon panier (H2)                                         │
├──────────────────────────────────────────────────────────┤
│  ┌─ Articles ───────────────────────────┬─ Résumé ───┐  │
│  │  [Item 1]                    € 290   │             │  │
│  │  [Item 2]                    € 450   │  Sous-total │  │
│  │  [Item 3]                    € 120   │  € 860      │  │
│  │                                       │             │  │
│  │  [Continuer mes achats]              │  Livraison  │  │
│  │                                       │  Offerte     │  │
│  │                                       │             │  │
│  │                                       │  [Commander] │  │
│  └───────────────────────────────────────┘             │  │
│                                           └─────────────┘  │
├─ FOOTER ───────────────────────────────────────────────────┤
└────────────────────────────────────────────────────────────┘
```

- 2 colonnes : articles (gauche, 2/3) + résumé (droite, 1/3)
- Chaque item : thumbnail + détails + quantité + prix + supprimer
- Résumé : sous-total, livraison, total, bouton "Commander"

### 6.5 Checkout (`checkout.html`)

```
┌──────────────────────────────────────────────────────────┐
│  Paiement sécurisé (H2)                                  │
├──────────────────────────────────────────────────────────┤
│  ┌─ Coordonnées ───────────────┬─ Récapitulatif ──────┐  │
│  │  Email                       │  [Item 1]    € 290   │  │
│  │  [input]                     │  [Item 2]    € 450   │  │
│  │                              │                       │  │
│  │  Livraison                   │  Sous-total  € 740   │  │
│  │  Prénom NOM                  │  Livraison   Offerte │  │
│  │  [input]                     │  Total       € 740   │  │
│  │  Adresse                     │                       │  │
│  │  [textarea]                  │  [Payer]              │  │
│  │  Ville / Code postal         │                       │  │
│  │  [input] [input]             └───────────────────────┘  │
│  │                              │
│  │  Paiement                    │
│  │  ████ ████ ████ ████        │
│  │  MM/AA  CVC  [Payer]        │
│  └──────────────────────────────┘
├─ FOOTER (minimal) ────────────────────────────────────────┤
└────────────────────────────────────────────────────────────┘
```

- Layout 2 colonnes, formulaire à gauche, récapitulatif à droite
- Design épuré, pas de distractions, pas de navigation principale
- Bouton "Payer" accent filled, large
- Éléments de formulaire : border fine, focus ring accent

### 6.6 Confirmation (`confirmation.html`)

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│    ✓  Commande confirmée                                 │
│                                                          │
│    Merci, [Prénom] !                                      │
│    Nous vous avons envoyé un email de confirmation.      │
│                                                          │
│    Numéro de commande : MAISON-2025-0042                 │
│                                                          │
│    ┌─────────────────────────────────────────────────┐   │
│    │  Récapitulatif de la commande                   │   │
│    │  [Item 1]                    € 290              │   │
│    │  [Item 2]                    € 450              │   │
│    │  ─────────────────────────                     │   │
│    │  Total                       € 740              │   │
│    │  Adresse de livraison                           │   │
│    │  123 Rue de...                                  │   │
│    └─────────────────────────────────────────────────┘   │
│                                                          │
│    [Continuer mes achats]                                │
│                                                          │
├─ FOOTER (minimal) ────────────────────────────────────────┤
└────────────────────────────────────────────────────────────┘
```

- Centré, beaucoup d'espace blanc
- Icône de succès (checkmark sobre, pas de confettis)
- Card récapitulative avec bordure fine
- Navigation : retour à la boutique

### 6.7 Lookup / Order Search (`lookup.html`)

```
┌──────────────────────────────────────────────────────────┐
│  Suivre ma commande (H2)                                 │
│                                                          │
│  Saisissez votre numéro de commande et votre email.      │
│                                                          │
│  Numéro de commande                                      │
│  [________________________]                              │
│                                                          │
│  Email                                                   │
│  [________________________]                              │
│                                                          │
│  [Rechercher]                                            │
│                                                          │
│  ── Résultat ──────────────────────────────────────────  │
│  (affiché après soumission)                              │
│                                                          │
│  Commande MAISON-2025-0042                               │
│  Statut : En préparation                                 │
│  [Détail] [Annuler]                                      │
│                                                          │
├─ FOOTER ─────────────────────────────────────────────────┤
└──────────────────────────────────────────────────────────┘
```

- Formulaire centré, un bloc
- Résultat affiché dans la même page
- Design cohérent avec le reste

---

## 7. Layout Responsive (Intrinsic Design)

> **Aucun `@media` query dans les composants.** La responsivité est gérée par :
> - `clamp()` pour les tailles de typographie
> - `grid-template-columns: repeat(auto-fill, minmax(Npx, 1fr))` pour les grilles
> - `flex-wrap: wrap` pour les rangées de filtres, navbar
> - `container-type: inline-size` + `@container` pour les cartes (si nécessaire)
> - `width: 100%; max-width: var(--content-max, 1440px); margin-inline: auto;` pour le layout global

### Layout global

```css
:root {
  --content-max: 1440px;
  --content-narrow: 720px;  /* pour confirmation, lookup */
  --content-gutter: clamp(1rem, 3vw, 3rem);
}

.maison-layout {
  width: 100%;
  max-width: var(--content-max);
  margin-inline: auto;
  padding-inline: var(--content-gutter);
}

.maison-layout--narrow {
  max-width: var(--content-narrow);
}
```

### Grille produits
```css
.maison-product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-lg);
}
```

### 2 colonnes (produit, checkout)
```css
.maison-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-xl);
}
/* Passe en 1 colonne quand la vue est < 720px — mais sans @media,
   on utilise un container query ou auto-fit */
```

---

## 8. Fichiers à produire

> Tous les fichiers sont dans le répertoire du template e-commerce Go existant.
> Le CSS est divisé en trois fichiers chargés dans `<head>` via `<link>`.

### 8.1 CSS

| Fichier | Contenu |
|---|---|
| **`static/css/tokens.css`** | Design tokens (`:root`). Couleurs, espacements, typographie, ombres, rayons. Premier fichier chargé. |
| **`static/css/storefront.css`** | Tous les styles des pages publiques. Composants + layouts + pages. Organisé en sections commentées. |
| **`static/css/admin.css`** | Styles pour l'admin (commandes, clients). Sobriété similaire mais adapté à un usage bureau (tableaux, formulaires denses). |

### 8.2 Templates Go (html/template)

| Fichier | Description |
|---|---|
| **`templates/base.html`** | Template de base : `<!DOCTYPE html>`, `<head>` (meta, CSS, titre), `<body>`, footer, blocks `{{block "content" .}}`. Définit le layout global. |
| **`templates/home.html`** | Page d'accueil : hero, collections, valeurs, newsletter. |
| **`templates/products.html`** | Page collection/listing : breadcrumb, header, filtres, grille, pagination. |
| **`templates/product.html`** | Page détail produit : 2 colonnes, image principale/miniatures, infos, variantes, CTA, recommandations. |
| **`templates/cart.html`** | Panier : liste articles, résumé, CTA. |
| **`templates/checkout.html`** | Paiement : formulaire coordonnées + CB + récapitulatif. |
| **`templates/confirmation.html`** | Confirmation de commande. |
| **`templates/lookup.html`** | Recherche de commande par email + numéro. |

### 8.3 Ordre de chargement CSS

```html
<link rel="stylesheet" href="/static/css/tokens.css">
<link rel="stylesheet" href="/static/css/storefront.css">
```

---

## 9. Don't's (Règles strictes)

| ❌ Interdit | ✅ Alternative |
|---|---|
| `#fff`, `#000`, `white`, `black` | `var(--color-bg)`, `var(--color-text)` |
| `rgb(255 255 255)`, `rgb(0 0 0)` | `oklch(...)` tokens uniquement |
| `rgba(0,0,0,0.5)` pour les ombres | `oklch(0 0 0 / 0.5)` |
| `@media (max-width: ...)` dans les composants | `clamp()`, `auto-fill`, `minmax()`, `flex-wrap` |
| Hamburger menu / mobile toggle | Liens visibles, adaptés par wrap et taille |
| JavaScript requis pour le layout | Le layout doit fonctionner sans JS (JS seulement pour interactions : clic, ajout panier, accordéon filtres) |
| Slider / carousel automatique | Hero statique ou navigation manuelle par flèches |
| `!important` dans le CSS | Spécificité maîtrisée par BEM ou nesting |
| Bordures noires épaisses | `1px solid var(--color-border)` |
| Images overflow sans `object-fit` | Toujours `object-fit: cover` sur les images de produit |
| `font-weight: bold` (mot-clé) | `var(--weight-bold)` ou `700` |
| Couleurs hex dans les templates | Utiliser les variables CSS, jamais de couleur hardcodée dans les templates |
| Police par défaut du navigateur | `var(--font-body)` |
| Texte blanc sur fond clair | `var(--color-bg)` n'est jamais blanc pur, mais texte clair sur fond clair = illisible. `var(--color-text)` sur `var(--color-bg)` uniquement |

---

## 10. Notes d'implémentation

### État initial (template e-commerce existant)
- Go 1.27rc1, html/template SSR, modernc.org/sqlite
- Template en anglais → tout traduire en français
- Accent vert actuel `#1f5e46` → remplacer par `var(--color-accent)` or/champagne
- Palette hex existante → remplacer par tokens oklch
- Supprimer tout CSS en dur dans les templates

### Migration palette
```
Avant (vert)                →  Après (or/champagne)
#1f5e46 (accent)            →  oklch(0.72 0.11 68)
#ffffff (fond)              →  oklch(0.97 0.008 90)
#000000 (texte)             →  oklch(0.13 0.02 270)
#f5f5f5 (surface)           →  oklch(0.94 0.012 85)
#e0e0e0 (bordure)           →  oklch(0.85 0.012 80)
```

### Police par défaut (fallback stack)
Le template existant doit héberger ou lier Google Fonts. Si inaccessible :
```css
--font-display: 'Playfair Display', Georgia, 'Times New Roman', serif;
--font-body: Inter, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
```

### SVG et icônes
- Recherche : SVG loupe (ligne fine, 18×18)
- Panier : SVG sac (ligne fine, 18×18)
- Compte : SVG silhouette (ligne fine, 18×18)
- Favoris : SVG cœur (outline → filled au clic)
- Supprimer : SVG ✕
- Chevron : SVG flèche simple (filtrer, pagination)

---

## 11. Checklist frontend

- [ ] `tokens.css` — toutes les variables OKLCH, espacements, typographie, ombres
- [ ] `storefront.css` — navbar, hero, product card, buttons, filters, variant list, cart item, footer
- [ ] `base.html` — structure HTML, chargement CSS, blocks de contenu
- [ ] Traduction FR de tous les textes du template (anglais → français)
- [ ] Remplacement de toutes les couleurs hex par des `var(--color-*)`
- [ ] Suppression de tous les `@media` dans les composants
- [ ] Suppression du hamburger menu
- [ ] Ajustement des images au ratio 3:4
- [ ] Aucun `#fff`, `#000` dans tout le projet (vérifier avec `grep -rn '#fff\|#000\|white\|black' static/css/ templates/`)
- [ ] Responsive vérifié sans redimensionnement de fenêtre (intrinsic)
- [ ] `home.html` — hero, collections, valeurs, newsletter
- [ ] `products.html` — breadcrumb, header, filtres, grille, pagination
- [ ] `product.html` — 2 colonnes, sticky sidebar, variantes, CTA
- [ ] `cart.html` — 2 colonnes, items, résumé
- [ ] `checkout.html` — formulaire + récap
- [ ] `confirmation.html` — centré, sobre, récapitulatif
- [ ] `lookup.html` — formulaire + résultats

---

**STATUS: Brief complete — frontend-engineer can start building.**
