package model

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// RidgeRegression Ridge 回归模型
// beta = (X'X + lambda*I)^-1 X'y
type RidgeRegression struct {
	Lambda float64
	Coefs  []float64 // 回归系数（含截距项 coefs[0]）
	R2     float64
	MAPE   float64
}

// Fit 训练 Ridge 回归
// X: [n_samples x n_features]，y: [n_samples]
func (r *RidgeRegression) Fit(X [][]float64, y []float64) error {
	n := len(y)
	if n == 0 || len(X) != n {
		return fmt.Errorf("X 和 y 长度不匹配")
	}
	p := len(X[0]) + 1 // +1 for intercept

	// 构建设计矩阵（添加截距列）
	data := make([]float64, n*p)
	for i, row := range X {
		data[i*p] = 1.0
		for j, v := range row {
			data[i*p+j+1] = v
		}
	}
	Xm := mat.NewDense(n, p, data)
	ym := mat.NewVecDense(n, y)

	// X'X
	var XtX mat.Dense
	XtX.Mul(Xm.T(), Xm)

	// X'X + lambda*I
	for i := 0; i < p; i++ {
		XtX.Set(i, i, XtX.At(i, i)+r.Lambda)
	}

	// X'y
	var Xty mat.VecDense
	Xty.MulVec(Xm.T(), ym)

	// 求解线性方程组
	var beta mat.VecDense
	if err := beta.SolveVec(&XtX, &Xty); err != nil {
		return fmt.Errorf("求解回归系数失败: %w", err)
	}

	r.Coefs = make([]float64, p)
	for i := 0; i < p; i++ {
		r.Coefs[i] = beta.AtVec(i)
	}

	r.R2 = computeR2(Xm, ym, &beta)
	r.MAPE = computeMAPE(Xm, ym, &beta)
	return nil
}

// Predict 给定特征矩阵预测
func (r *RidgeRegression) Predict(X [][]float64) ([]float64, error) {
	if len(r.Coefs) == 0 {
		return nil, fmt.Errorf("模型未训练")
	}
	preds := make([]float64, len(X))
	for i, row := range X {
		val := r.Coefs[0] // intercept
		for j, v := range row {
			if j+1 < len(r.Coefs) {
				val += r.Coefs[j+1] * v
			}
		}
		preds[i] = val
	}
	return preds, nil
}

func computeR2(X *mat.Dense, y *mat.VecDense, beta *mat.VecDense) float64 {
	n, _ := X.Dims()
	var yHat mat.VecDense
	yHat.MulVec(X, beta)

	var yMean float64
	for i := 0; i < n; i++ {
		yMean += y.AtVec(i)
	}
	yMean /= float64(n)

	var ssTot, ssRes float64
	for i := 0; i < n; i++ {
		diff := y.AtVec(i) - yMean
		ssTot += diff * diff
		resid := y.AtVec(i) - yHat.AtVec(i)
		ssRes += resid * resid
	}
	if ssTot == 0 {
		return 1.0
	}
	return 1.0 - ssRes/ssTot
}

func computeMAPE(X *mat.Dense, y *mat.VecDense, beta *mat.VecDense) float64 {
	n, _ := X.Dims()
	var yHat mat.VecDense
	yHat.MulVec(X, beta)

	var mape float64
	count := 0
	for i := 0; i < n; i++ {
		actual := y.AtVec(i)
		if actual == 0 {
			continue
		}
		mape += math.Abs((actual - yHat.AtVec(i)) / actual)
		count++
	}
	if count == 0 {
		return 0
	}
	return mape / float64(count)
}
