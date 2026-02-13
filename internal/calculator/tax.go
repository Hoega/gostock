package calculator

import (
	"math"

	"github.com/Hoega/gostock/internal/model"
)

// PFURate is the French flat tax rate (30%)
const PFURate = 0.30

// ComputeStockSaleSummary calculates summary statistics for stock sales.
func ComputeStockSaleSummary(sales []model.StockSale, purchases []model.StockPurchase) model.StockSaleSummary {
	summary := model.StockSaleSummary{
		Sales:     sales,
		Purchases: purchases,
	}

	for _, sale := range sales {
		gain := sale.CapitalGain()
		if gain >= 0 {
			summary.TotalPV += gain
		} else {
			summary.TotalMV += math.Abs(gain)
		}
	}

	summary.TotalPV = math.Round(summary.TotalPV*100) / 100
	summary.TotalMV = math.Round(summary.TotalMV*100) / 100
	summary.NetResult = math.Round((summary.TotalPV-summary.TotalMV)*100) / 100

	// Taxable base is positive net result only
	if summary.NetResult > 0 {
		summary.TaxableBase = summary.NetResult
	}

	summary.PFUAmount = math.Round((summary.TaxableBase*PFURate)*100) / 100

	return summary
}

// ComputeCryptoSaleSummary calculates summary statistics for crypto sales.
func ComputeCryptoSaleSummary(sales []model.CryptoSale) model.CryptoSaleSummary {
	summary := model.CryptoSaleSummary{
		Sales: sales,
	}

	for _, sale := range sales {
		// Total cessions = sum of all sale proceeds (case 3AN)
		summary.TotalCessions += sale.SaleProceeds()

		gain := sale.CapitalGain()
		if gain >= 0 {
			summary.TotalPV += gain
		} else {
			summary.TotalMV += math.Abs(gain)
		}
	}

	summary.TotalCessions = math.Round(summary.TotalCessions*100) / 100
	summary.TotalPV = math.Round(summary.TotalPV*100) / 100
	summary.TotalMV = math.Round(summary.TotalMV*100) / 100
	summary.NetResult = math.Round((summary.TotalPV-summary.TotalMV)*100) / 100

	// Case 3BN is positive net result only
	if summary.NetResult > 0 {
		summary.TaxableBase = summary.NetResult
	}

	summary.PFUAmount = math.Round((summary.TaxableBase*PFURate)*100) / 100

	return summary
}

// ComputeTaxYearSummary computes full summary for a tax year.
func ComputeTaxYearSummary(year int, stockSales []model.StockSale, cryptoSales []model.CryptoSale) model.TaxYearSummary {
	stockSummary := ComputeStockSaleSummary(stockSales, nil)
	cryptoSummary := ComputeCryptoSaleSummary(cryptoSales)

	summary := model.TaxYearSummary{
		Year: year,

		StockSales:       stockSales,
		StockTotalPV:     stockSummary.TotalPV,
		StockTotalMV:     stockSummary.TotalMV,
		StockNetResult:   stockSummary.NetResult,
		StockTaxableBase: stockSummary.TaxableBase,

		CryptoSales:         cryptoSales,
		CryptoTotalCessions: cryptoSummary.TotalCessions,
		CryptoTotalPV:       cryptoSummary.TotalPV,
		CryptoTotalMV:       cryptoSummary.TotalMV,
		CryptoNetResult:     cryptoSummary.NetResult,
		CryptoTaxableBase:   cryptoSummary.TaxableBase,
	}

	summary.TotalTaxableBase = summary.StockTaxableBase + summary.CryptoTaxableBase
	summary.PFUAmount = math.Round((summary.TotalTaxableBase*PFURate)*100) / 100

	return summary
}

// Compute2086Fields returns the values for French 2086 form fields.
// Returns: case3AN (total cessions), case3BN (taxable plus-value)
func Compute2086Fields(sales []model.CryptoSale) (case3AN, case3BN float64) {
	summary := ComputeCryptoSaleSummary(sales)
	return summary.TotalCessions, summary.TaxableBase
}
