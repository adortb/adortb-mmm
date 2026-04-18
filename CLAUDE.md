# CLAUDE.md — adortb-mmm

## 项目角色

十二期服务：媒体组合模型（MMM）。离线训练为主，每周或每月刷新一次模型，用于预算策略决策而非实时竞价。

## 关键函数与复杂度

| 函数 | 文件 | 复杂度 | 说明 |
|------|------|--------|------|
| `GeometricAdstock` | `model/adstock.go` | O(T) | 几何衰减，按时间步顺序递推 |
| `WeightedAdstock` | `model/adstock.go` | O(T·W) | 自定义权重窗口，W=窗口长度 |
| `HillSaturation` | `model/saturation.go` | O(1) | Hill函数单点计算 |
| `BatchHillSaturation` | `model/saturation.go` | O(T) | 批量Hill变换 |
| `Fit` | `model/fit.go` | O(C·L·A·T·K²) | 网格搜索，C=渠道，L=lambda候选，A=alpha候选，K²=Ridge拟合 |
| `Optimize` | `optimizer/budget.go` | O(I·C·T) | 投影梯度上升，I=迭代次数（500），C=渠道 |
| `RidgeRegression.Fit` | `model/regression.go` | O(T·C²) | Ridge OLS |

## 模型训练规范

- 最少数据量：10个周期（`Fit` 函数强制检查），建议 ≥ 52 周
- 验证集比例：`ValFraction=0.2`（后20%作为验证集）
- 超参搜索独立进行（per channel），避免渠道间干扰
- `estimateGamma` 用 adstocked 均值作为半饱和点初始估计，适合首次拟合

## Adstock 变换注意事项

- `GeometricAdstock` 要求按**时间顺序**（从 t=0 到 t=T-1），不得打乱
- `lambda=0` 表示无历史效果（即时效果），`lambda=1` 表示永久累积（需谨慎）
- `WeightedAdstock` 权重数组必须归一化（Σ weights = 1），否则尺度不一致

## Hill 饱和曲线注意事项

- `x <= 0` 时返回 0（零投入无效果）
- `alpha > 3` 会产生极端 S 曲线，通常不推荐
- `gamma` 从数据估计（均值法），可在后续版本改为参数化拟合

## 预算优化注意事项

- 梯度步长 `lr = defaultLR * TotalBudget / 100`，与预算规模联动
- 使用 `Constraints.Project` 做投影（约束满足），实现详见 `optimizer/constraints.go`
- 收敛判据：`|conv_new - conv_prev| < 1e-6 * conv_new`
- 最大迭代 500 次，一般 < 200 次收敛

## 训练数据格式

```go
type DataPoint struct {
    Spends []float64  // 每个渠道的支出，顺序与 Config.Channels 一致
    Target float64    // 目标指标（如 revenue / conversions）
}
```

渠道顺序**必须固定**且与 `FitConfig.Channels` 完全一致，否则特征矩阵对齐错误。

## 测试

```bash
go test -race ./...
go test -v ./internal/model/ -run TestAdstock
go test -v ./internal/model/ -run TestHillSaturation
go test -v ./internal/model/ -run TestFit
go test -v ./internal/optimizer/ -run TestOptimize
```

关键测试：
- `model/adstock_test.go` — 几何衰减递推正确性
- `model/saturation_test.go` — Hill函数单调性、边界值
- `model/fit_test.go` — 网格搜索找到合理参数
- `model/regression_test.go` — Ridge回归数值正确性
- `optimizer/budget_test.go` — 优化结果满足约束、比均匀分配更优
