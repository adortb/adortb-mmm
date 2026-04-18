# Architecture — adortb-mmm

## 系统概览

```
┌─────────────────────────────────────────────────────┐
│                   HTTP API Server                    │
│  /v1/fit   /v1/optimize   /v1/contribution           │
└──────────────┬──────────────────────────────────────┘
               │
    ┌──────────▼──────────┐
    │ Aggregator / Fetcher │  ClickHouse 周度数据
    └──────────┬──────────┘
               │
    ┌──────────▼──────────┐
    │    MMM Fit Engine    │  网格搜索 + Ridge回归
    └──────────┬──────────┘
               │
    ┌──────────▼──────────┐     ┌───────────────────┐
    │   FittedModel        │────►│  Budget Optimizer  │
    └──────────┬──────────┘     └────────┬──────────┘
               │                         │
    ┌──────────▼──────────┐     ┌────────▼──────────┐
    │  Contribution Report │     │  OptimizeResult    │
    └─────────────────────┘     └───────────────────┘
```

## 训练流程

```
周度花费/转化数据（ClickHouse）
    │
    ▼
┌────────────────────────────────────────────────────┐
│                   Fit(data, cfg)                    │
│                                                     │
│  Per-channel Grid Search:                          │
│  for ch in channels:                               │
│    for λ in [0.1,0.3,0.5,0.7,0.9]:               │
│      for α in [1.0,2.0,3.0]:                      │
│        ① GeometricAdstock(spend_ch, λ)            │
│        ② BatchHillSaturation(adstocked, α, γ̂)    │
│        ③ RidgeRegression.Fit(X_train, y_train)   │
│        ④ MAPE on val_set → 选最优 (λ*, α*)        │
│                                                     │
│  Final Model:                                       │
│    γ* = estimateGamma(data, ch, λ*)               │
│    X_full = build_feature_matrix(data, best_params)│
│    Ridge.Fit(X_full, y_full)                       │
│    val_MAPE = MAPE(Ridge.Predict(X_val), y_val)   │
└────────────────────────────────────────────────────┘
    │
    ▼
FittedModel { BestParams, RidgeRegression, ValidationMAPE }
```

## 推理流程（预测）

```
FittedModel + futureSpends
    │
    ▼
┌────────────────────────────────────────────────┐
│               FittedModel.Predict()             │
│                                                 │
│  for each week t:                              │
│    for each channel ch:                        │
│      ① adstocked[ch][t] = GeometricAdstock()  │
│      ② saturated[ch][t] = HillSaturation()    │
│    ③ X[t] = [saturated[ch1][t], ...]          │
│  ④ predictions = Ridge.Predict(X)             │
└────────────────────────────────────────────────┘
    │
    ▼
[]float64 (每周预测值)
```

## 预算优化流程

```
FittedModel + TotalBudget + Constraints
    │
    ▼
┌───────────────────────────────────────────────────┐
│            Projected Gradient Ascent               │
│                                                    │
│  初始化：均匀分配 alloc = TotalBudget / N          │
│  Project(alloc) → 满足约束                         │
│                                                    │
│  for iter in [0, 500):                            │
│    grad = numerical_gradient(FittedModel, alloc)   │
│          (有限差分 Δalloc_i = lr * 0.01)           │
│    alloc = alloc + lr * grad                       │
│    alloc = Project(alloc)   ← 投影到可行域         │
│    if |conv_new - conv_prev| < 1e-6: break        │
│                                                    │
│  lift = (finalConv - equalConv) / equalConv       │
└───────────────────────────────────────────────────┘
    │
    ▼
OptimizeResult { Allocation, ExpectedConversions, LiftVsEqual }
```

## 模型结构图

```
渠道支出序列
  SEM:     [s₁, s₂, ..., s_T]
  Social:  [s₁, s₂, ..., s_T]
  Display: [s₁, s₂, ..., s_T]
         │
         │ Adstock(λ*)
         ▼
  各渠道 adstock 序列
         │
         │ HillSaturation(α*, γ*)
         ▼
  饱和后特征矩阵 X (T × C)
         │
         │ Ridge Regression (β, β₀)
         ▼
  revenue[t] = β₀ + Σ_ch β_ch · X[t,ch]
```

## 数据输入输出

### 输入（ClickHouse 周度数据）

```sql
SELECT week, channel, SUM(spend) as spend, SUM(conversions) as target
FROM adortb_spend_log
WHERE week BETWEEN ? AND ?
GROUP BY week, channel
ORDER BY week
```

### 输出

**贡献分解：**
```json
{
  "baseline_contribution": 0.35,
  "channel_contributions": {
    "sem":     { "pct": 0.28, "marginal_roi": 3.2 },
    "social":  { "pct": 0.22, "marginal_roi": 2.1 },
    "display": { "pct": 0.15, "marginal_roi": 1.8 }
  }
}
```

**预算优化：**
```json
{
  "allocation": {
    "sem":     450000,
    "social":  320000,
    "display": 230000
  },
  "expected_conversions": 12500,
  "lift_vs_equal": 0.18
}
```

## 评估指标

| 指标 | 计算方式 | 目标 |
|------|----------|------|
| MAPE（验证集） | mean(|y-ŷ|/y) | < 20% |
| R² | 1 - SS_res/SS_tot | > 0.85 |
| 优化提升（LiftVsEqual） | (opt_conv - equal_conv)/equal_conv | > 0（说明优化有效）|

## 依赖关系

```
adortb-mmm
└── ClickHouse  （周度渠道花费 + 转化数据）
```
