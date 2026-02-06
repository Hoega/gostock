# GoStock - Documentation des fonctionnalités

GoStock est un simulateur de crédit immobilier français. Il permet d'analyser en détail un projet d'achat immobilier et de comparer différents scénarios.

## Table des matières

### Utilisation
1. [Paramètres d'entrée](#paramètres-dentrée)
2. [Résultats de la simulation](#résultats-de-la-simulation)

### Graphiques Donut
3. [Répartition du coût total](#graphique-donut--répartition-du-coût)
4. [Coûts irrécupérables](#graphique-donut--coûts-irrécupérables)

### Tableau
5. [Tableau d'amortissement](#tableau-damortissement)

### Graphiques de projection
6. [Légende des graphiques](#légende-des-graphiques)
7. [Rentabilité à la revente](#graphique-de-rentabilité-à-la-revente)
8. [Location vs Achat](#comparaison-location-vs-achat)
9. [Cash brut à la revente](#cash-brut-à-la-revente)
10. [Cash net à la revente](#cash-net-à-la-revente)

### Technique
11. [Fonctionnalités de l'interface](#fonctionnalités-de-linterface)
12. [Technologies utilisées](#technologies-utilisées)
13. [Formules mathématiques](#formules-mathématiques)

---

## Paramètres d'entrée

### Informations sur le bien

| Paramètre | Description |
|-----------|-------------|
| **Prix du bien** | Prix d'achat du bien immobilier en euros |
| **Apport personnel** | Montant de l'apport (le montant emprunté = prix - apport) |

### Conditions du prêt

| Paramètre | Description |
|-----------|-------------|
| **Taux d'intérêt annuel** | Taux nominal annuel du crédit (%) |
| **Durée** | Durée du prêt en années (1-30 ans) |
| **Date de début** | Mois et année de début du remboursement |
| **Taux assurance annuel** | Taux de l'assurance emprunteur sur le capital initial (%) |

### Frais

| Paramètre | Description |
|-----------|-------------|
| **Frais de notaire** | Pourcentage du prix du bien (boutons rapides : 7.5% ancien, 2.5% neuf) |
| **Frais d'agence (%)** | Pourcentage du prix du bien |
| **Frais d'agence (fixe)** | Montant fixe en euros (remplace le pourcentage si renseigné) |
| **Frais de dossier** | Frais bancaires fixes en euros |

### Dépenses liées au bien

| Paramètre | Description |
|-----------|-------------|
| **Travaux immédiats** | Enveloppe travaux/rénovation. S'ajoute au coût du projet |
| **Valorisation des travaux** | Coefficient de valorisation (70% par défaut). 1€ travaux = X% de valeur ajoutée |
| **Taxe foncière annuelle** | Impôt foncier annuel (coût irrécupérable) |
| **Charges de copropriété** | Charges mensuelles de copropriété (coût irrécupérable)

### Revenus (optionnel)

| Paramètre | Description |
|-----------|-------------|
| **Revenu mensuel net - Emprunteur 1** | Salaire net mensuel du premier emprunteur |
| **Revenu mensuel net - Emprunteur 2** | Salaire net mensuel du co-emprunteur |

### Comparaison location (optionnel)

| Paramètre | Description |
|-----------|-------------|
| **Loyer mensuel actuel** | Loyer actuellement payé en location |
| **Revalorisation annuelle du loyer** | Augmentation annuelle estimée du loyer (%) |
| **Rendement épargne annuel** | Taux de rendement si l'apport est placé (coût d'opportunité) |

---

## Résultats de la simulation

### Mensualités

- **Mensualité totale** : Somme de la part crédit et de l'assurance
- **Part crédit** : Mensualité hors assurance (capital + intérêts)
- **Part assurance** : Montant mensuel de l'assurance emprunteur

### Coûts

- **Coût total du crédit** : Intérêts + assurance sur toute la durée
- **Frais annexes** : Détail des frais de notaire, agence et dossier
- **Coût total du projet** : Prix + tous les frais + intérêts + assurance

### Analyse des revenus (si renseignés)

- **Taux d'effort** : Mensualité / revenus mensuels (alerte si > 35%)
- **Capacité d'emprunt max (HCSF)** : Montant maximum empruntable selon la règle des 35%
- **Revenus totaux sur 25 ans** : Projection des revenus sur 25 ans
- **Part du projet sur 25 ans** : Coût total / revenus 25 ans

### Règle HCSF des 35%

Le Haut Conseil de Stabilité Financière (HCSF) impose un taux d'effort maximum de 35% pour l'octroi de crédits immobiliers. Cela signifie que la mensualité totale (crédit + assurance) ne doit pas dépasser 35% des revenus nets mensuels.

**Calcul de la capacité d'emprunt maximale** :

```
Mensualité max = Revenus nets × 0.35
```

Le montant maximum empruntable est calculé en inversant la formule d'amortissement :

```
Capital max = Mensualité max / (t / (1 - (1+t)^(-n)) + taux_assurance_mensuel)
```

L'indicateur affiche également :
- La **marge** restante si le prêt est sous la limite
- Le **dépassement** si le prêt dépasse la capacité maximale

---

## Graphique Donut : Répartition du coût

### Description

Ce graphique en forme de donut montre la répartition du coût total du crédit entre les trois composantes principales.

### Composantes

| Couleur | Composante | Description |
|---------|------------|-------------|
| 🟣 Indigo | Capital | Montant emprunté (remboursé à la banque) |
| 🟡 Ambre | Intérêts | Coût des intérêts sur la durée du prêt |
| 🔴 Rose | Assurance | Coût total de l'assurance emprunteur |

### Interprétation

- Le **capital** représente généralement la plus grande part (c'est normal, c'est l'argent que vous empruntez réellement)
- Les **intérêts** dépendent du taux et de la durée du prêt
- L'**assurance** est souvent sous-estimée mais peut représenter une part significative
- Le tooltip affiche le montant et le pourcentage de chaque composante

### Tableau de détail

À côté du graphique, un tableau récapitule les montants exacts avec le total remboursé (capital + intérêts + assurance).

---

## Graphique Donut : Coûts irrécupérables

### Description

Ce graphique en forme de donut montre la répartition de tous les coûts irrécupérables, c'est-à-dire l'argent qui ne sera jamais récupéré à la revente du bien.

### Composantes

| Couleur | Composante | Description |
|---------|------------|-------------|
| 🔴 Rouge | Frais de notaire | Frais d'acquisition payés au notaire |
| 🟠 Orange | Frais d'agence | Commission de l'agent immobilier |
| 🟡 Jaune | Frais de dossier | Frais bancaires de montage du prêt |
| 🟡 Ambre | Intérêts | Coût total des intérêts sur la durée |
| 🔴 Rose | Assurance | Coût total de l'assurance emprunteur |
| 🟣 Violet | Taxe foncière | Impôt foncier cumulé sur la durée |
| 🟣 Indigo | Charges copro | Charges de copropriété cumulées |

### Ce qui n'est PAS un coût irrécupérable

- **Capital remboursé** : Récupéré car il réduit le capital restant dû
- **Travaux** : Partiellement récupérés selon le coefficient de valorisation
- **Apport personnel** : Récupéré à la revente (équité)

> **Note sur les travaux** : Si le coefficient de valorisation est de 70%, alors 30% du coût des travaux est effectivement un coût irrécupérable. Cette nuance n'est pas reflétée dans ce graphique qui utilise le coefficient pour la valorisation du bien.

### Interprétation

- Ce graphique montre "l'argent brûlé" dans l'opération immobilière
- Plus les intérêts dominent, plus le taux est élevé ou la durée longue
- Les frais de notaire sont souvent le 2ème poste de coût irrécupérable
- Utile pour comparer avec le coût cumulé d'une location équivalente

### Tableau de détail

À côté du graphique, un tableau détaille chaque composante avec :
- Le montant individuel de chaque coût
- Le total irrécupérable sur toute la durée du prêt

---

## Tableau d'amortissement

Tableau mensuel détaillant pour chaque échéance :

| Colonne | Description |
|---------|-------------|
| **Date** | Mois et année de l'échéance |
| **Mensualité** | Montant de la mensualité (hors assurance) |
| **Capital** | Part du capital remboursé |
| **Intérêts** | Part des intérêts |
| **Assurance** | Montant de l'assurance |
| **Capital restant dû** | Solde restant à rembourser |

---

## Légende des graphiques

Tous les graphiques utilisent un code couleur cohérent pour représenter les différents scénarios de valorisation annuelle du bien :

| Couleur | Scénario | Interprétation |
|---------|----------|----------------|
| 🔴 Rouge | -3% / an | Scénario pessimiste (crise immobilière) |
| 🟠 Orange | -1% / an | Scénario légèrement négatif |
| ⚫ Gris | 0% / an | Scénario stagnant (pas de valorisation) |
| 🟢 Vert clair | +1% / an | Scénario légèrement positif |
| 🟢 Vert | +2% / an | Scénario optimiste modéré |
| 🔵 Bleu | +3% / an | Scénario très optimiste |

> **Note** : Le scénario 0% est souvent le plus réaliste sur le long terme, une fois l'inflation prise en compte.

---

## Graphique de rentabilité à la revente

### Description

Ce graphique montre la **plus-value nette** en cas de revente du bien selon différents scénarios de valorisation annuelle.

### Calcul

```
Plus-value nette = Valeur du bien - Capital restant dû - Total dépensé
```

Où :
- **Valeur du bien** = (Prix + Travaux) × (1 + taux)^années
- **Total dépensé** = Apport + Frais notaire + Frais agence + Frais dossier + Travaux + Cumul des mensualités + Taxe foncière cumulée + Charges copro cumulées

### Éléments du graphique

- **Axe X** : Durée de détention (en années)
- **Axe Y** : Plus-value nette (en euros)
- **Ligne pointillée noire** : Seuil de rentabilité (y = 0)
- **6 courbes colorées** : Un scénario par taux de valorisation

### Interprétation

- **Au-dessus de 0** : La vente génère un profit
- **En-dessous de 0** : La vente génère une perte
- Le point où une courbe croise la ligne pointillée indique le **nombre d'années minimum** pour être rentable dans ce scénario

---

## Comparaison Location vs Achat

### Description

Ce graphique compare le **patrimoine net** accumulé selon que vous restiez locataire ou que vous achetiez. **Plus haut = mieux.** Quand une courbe "Acheteur" dépasse la courbe "Locataire", l'achat devient plus avantageux.

### Coût d'opportunité de l'apport

Le simulateur intègre le **coût d'opportunité** de l'apport personnel :

- Si vous restez locataire, vous ne dépensez pas votre apport
- Cet apport peut être placé (ex: PEA à 7%, Livret A à 3%, assurance-vie à 4%)
- Le patrimoine du locataire = valeur du placement - loyers cumulés

> **Pourquoi c'est important ?** Sans cette prise en compte, la comparaison est biaisée en faveur de l'achat. L'argent immobilisé dans un apport a un coût d'opportunité réel.

### Calculs

**Patrimoine locataire** :
```
Patrimoine = Valeur placement - Cumul loyers
           = Apport × (1 + taux_épargne)^N - Σ loyers
```

**Patrimoine acheteur** (si revente) :
```
Patrimoine = Équité - Coûts irrécupérables
           = (Valeur bien - Capital restant) - (Frais + Intérêts + Assurance + Taxes + Charges)
```

### Éléments du graphique

- **Axe X** : Durée de détention (en années)
- **Axe Y** : Patrimoine net (en euros) - **Plus haut = mieux**
- **Ligne noire épaisse** : Patrimoine du locataire
- **6 courbes colorées** : Patrimoine de l'acheteur par scénario
- **Ligne pointillée violette** : Niveau de l'apport initial
- **Ligne pointillée noire** : Seuil zéro

### Interprétation

- Quand une courbe "Acheteur" **dépasse** la ligne noire "Locataire" : l'achat devient plus avantageux
- Plus le rendement épargne est élevé, plus la location reste compétitive longtemps
- Plus l'apport est important, plus le coût d'opportunité pèse
- Les valeurs peuvent être négatives (vous avez "perdu" de l'argent vs votre situation initiale)

### Exemple de lecture

Avec un apport de 50 000 € :
- **Locataire après 10 ans** : Placement vaut 81 445 €, loyers payés 120 000 € → Patrimoine = -38 555 €
- **Acheteur après 10 ans (+2%/an)** : Équité 180 000 €, coûts irrécupérables 85 000 € → Patrimoine = 95 000 €
- → L'achat est plus avantageux dans ce scénario

> **Attention** : Ce graphique ne prend pas en compte les avantages non financiers de la propriété (sécurité, liberté de modifications, etc.)

---

## Cash brut à la revente

### Description

Ce graphique montre le **montant d'argent récupéré** en cas de vente du bien, après remboursement du capital restant dû à la banque. C'est la somme que vous recevez concrètement lors de la vente.

### Calcul

```
Cash brut = Valeur du bien - Capital restant dû
```

### Éléments du graphique

- **Axe X** : Durée de détention (en années)
- **Axe Y** : Cash récupéré (en euros)
- **Ligne pointillée noire** : Seuil de rentabilité (y = 0)
- **6 courbes colorées** : Cash brut par scénario de valorisation

### Interprétation

- Représente l'argent qui revient dans votre poche après la vente
- **Ne tient pas compte** des frais et coûts engagés (ce n'est pas un profit)
- Utile pour évaluer la **liquidité disponible** en cas de revente
- Les courbes montent car le capital restant dû diminue avec le temps
- Si une courbe descend, c'est que la dévalorisation du bien est plus rapide que le remboursement du capital

---

## Cash net à la revente

### Description

Ce graphique montre le **profit réel** en cas de vente, après déduction de tous les coûts irrécupérables. C'est l'indicateur le plus pertinent pour évaluer la rentabilité d'un investissement immobilier.

### Calcul

```
Cash net = Cash brut - Coûts irrécupérables
```

**Coûts irrécupérables** = Frais de notaire + Frais d'agence + Frais de dossier + Intérêts cumulés + Assurance cumulée + Taxe foncière cumulée + Charges copropriété cumulées

> **Note** : Le remboursement du capital n'est PAS un coût irrécupérable car il réduit le capital restant dû.

> **Note** : Les travaux ne sont PAS un coût irrécupérable car ils valorisent le bien.

### Éléments du graphique

- **Axe X** : Durée de détention (en années)
- **Axe Y** : Cash récupéré (en euros)
- **Ligne pointillée noire (y=0)** : Seuil de rentabilité
- **Ligne pointillée violette** : Niveau de l'apport initial
- **6 courbes colorées** : Cash net par scénario de valorisation

### Lignes de référence

| Ligne | Signification |
|-------|---------------|
| **Seuil y=0** | En dessous, vous perdez de l'argent |
| **Ligne apport** | Au-dessus, vous avez récupéré votre mise + un bénéfice |

### Interprétation

- **Au-dessus de l'apport** : 🎉 Vous récupérez votre apport initial + un bénéfice net
- **Entre 0 et l'apport** : ✅ Vous êtes rentable mais n'avez pas encore récupéré tout votre apport
- **En-dessous de 0** : ❌ Vous perdez de l'argent sur l'opération

### Exemple de lecture

Si après 10 ans avec un scénario +1%/an, la courbe indique 25 000 € et que votre apport était de 50 000 € :
- Vous récupérez 25 000 € de cash net
- Vous avez "perdu" 25 000 € par rapport à votre apport initial
- Mais si vous étiez resté locataire, vous auriez peut-être dépensé plus en loyers

---

## Fonctionnalités de l'interface

### Calcul en temps réel

Le simulateur recalcule automatiquement les résultats lorsque vous modifiez un champ :

- **Debounce de 500ms** : Le calcul se déclenche 500ms après la dernière modification
- **Indicateur de chargement** : Un spinner s'affiche pendant le calcul
- **Pas besoin de cliquer** : Les graphiques se mettent à jour automatiquement

> **Technique** : Utilisation de `hx-trigger="keyup changed delay:500ms"` avec HTMX

### Boutons rapides

- **Frais de notaire** : Boutons "Ancien" (7.5%) et "Neuf" (2.5%) pour remplir rapidement

### Indicateurs visuels

- **Taux d'effort > 35%** : Affiché en rouge avec un avertissement
- **Capacité d'emprunt** : Affiche la marge (vert) ou le dépassement (rouge)
- **Seuil de rentabilité** : Ligne pointillée sur les graphiques

---

## Technologies utilisées

| Composant | Technologie | Usage |
|-----------|-------------|-------|
| **Backend** | Go 1.21+ | Serveur HTTP, calculs |
| **Router** | Chi v5 | Routage REST |
| **Frontend** | HTMX 2.0 | Interactions dynamiques sans JavaScript |
| **Graphiques** | Chart.js 4.x | Visualisations (line, doughnut) |
| **Annotations** | chartjs-plugin-annotation | Lignes de référence sur les graphiques |
| **Style** | Tailwind CSS | Design responsive |
| **Templates** | html/template | Rendu HTML côté serveur |

### Architecture HTMX

Le simulateur utilise une architecture "HTML over the wire" :

1. Le formulaire envoie une requête POST via HTMX
2. Le serveur calcule et renvoie des fragments HTML
3. HTMX remplace le contenu de `#results` avec la réponse
4. Les scripts Chart.js s'exécutent pour générer les graphiques

Avantages :
- Pas de framework JavaScript complexe
- État géré côté serveur
- Temps de chargement initial rapide
- SEO-friendly

---

## Formules mathématiques

### Mensualité (annuité constante)

```
M = C × t / (1 - (1+t)^(-n))
```

Où :
- M = mensualité
- C = capital emprunté
- t = taux mensuel (taux annuel / 12)
- n = nombre de mensualités

### Assurance mensuelle

```
Assurance = (Taux assurance annuel / 12) × Capital initial
```

### Valorisation du bien

```
Valeur ajoutée travaux = Travaux × Coefficient valorisation
Valeur année N = (Prix initial + Valeur ajoutée travaux) × (1 + taux valorisation)^N
```

**Coefficient de valorisation des travaux** (70% par défaut) :

| Type de travaux | Coefficient suggéré | Explication |
|-----------------|---------------------|-------------|
| Toiture, façade | 50-70% | Maintient la valeur plus qu'il ne l'augmente |
| Isolation, chauffage | 60-80% | Améliore le DPE, valorisation variable |
| Cuisine, salle de bain | 80-100% | Forte valeur perçue par les acheteurs |
| Extension, surélévation | 90-120% | Peut créer plus de valeur que le coût |

> **Important** : Le coût des travaux reste à 100% dans les dépenses, seule la valorisation du bien est affectée par le coefficient. C'est plus réaliste : vous dépensez 10 000€, mais le bien ne prend que 7 000€ de valeur.

### Capacité d'emprunt maximale (HCSF)

La règle HCSF impose un taux d'effort maximum de 35% :

```
Mensualité max = Revenus nets mensuels × 0.35
```

Le capital maximum empruntable est calculé en inversant la formule de mensualité :

```
Capital max = Mensualité max / (t / (1 - (1+t)^(-n)) + taux_assurance_mensuel)
```

Où :
- t = taux d'intérêt mensuel
- n = nombre de mensualités
- taux_assurance_mensuel = taux assurance annuel / 12

### Coûts irrécupérables

```
Coûts irrécupérables = Frais notaire + Frais agence + Frais dossier
                     + Intérêts cumulés + Assurance cumulée
                     + Taxe foncière × années + Charges copro × mois
```

### Cash net à la revente

```
Cash net = (Valeur bien - Capital restant dû) - Coûts irrécupérables
```

### Taux d'effort

```
Taux d'effort = (Mensualité + Assurance) / Revenus nets × 100
```

### Patrimoine net (Location vs Achat)

**Patrimoine locataire** :
```
Valeur placement = Apport × (1 + taux_épargne)^N
Patrimoine locataire = Valeur placement - Cumul loyers
```

**Patrimoine acheteur** :
```
Équité = Valeur bien - Capital restant dû
Coûts irrécupérables = Frais + Intérêts + Assurance + Taxes + Charges
Patrimoine acheteur = Équité - Coûts irrécupérables
```

Cette comparaison montre ce que chaque personne possède réellement après N années, en tenant compte du coût d'opportunité de l'apport immobilisé.
