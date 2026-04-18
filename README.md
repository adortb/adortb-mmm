# adortb-mmm

十二期服务。媒体组合模型（Media Mix Model），量化各广告渠道对业务指标的贡献，并通过预算优化最大化 ROI。

## 算法概述

### Adstock 变换（广告留存效应）

广告效果不会立即消失，而是随时间衰减，称为 Adstock（ad carryover）：

**几何衰减 Adstock**：
```
adstock[t] = spend[t] + λ · adstock[t-1]

λ ∈ [0,1]：衰减率（lambda），越大效果持续越久
```

**加权 Adstock**（自定义衰减窗口）：
```
adstock[t] = Σ_w weight[w] · spend[t-w]
```

### Hill 饱和曲线变换

广告效果存在边际递减规律，使用 Hill 函数建模饱和效应：

```
saturation(x) = x^α / (x^α + γ^α)

α (alpha)：形状参数，控制 S 曲线陡峭度（典型值 1~3）
γ (gamma)：半饱和点，即效果为最大值 50% 时的投入量
           γ 从训练数据自动估计（以 adstocked 均值作为初始估计）
```

Michaelis-Menten 饱和（α=1 特例）：
```
saturation(x) = x / (x + γ)
```

### Ridge 回归

对变换后的渠道特征做 Ridge（L2 正则化）线性回归：

```
revenue = β₀ + Σ_ch β_ch · hill(adstock(spend_ch))

损失函数：||y - Xβ||² + λ||β||²
```

### 超参数搜索

对每个渠道独立网格搜索 `(lambda, alpha)` 最优组合：
- `Lambdas`: [0.1, 0.3, 0.5, 0.7, 0.9]
- `Alphas`: [1.0, 2.0, 3.0]
- 评估指标：验证集 MAPE（训练集 80% / 验证集 20%）

### 预算优化

使用**投影梯度上升**最大化预期转化：

```
max Σ predicted_conversions(alloc)
subject to:
  Σ alloc_ch = TotalBudget        （总预算约束）
  MinBudget_ch ≤ alloc_ch ≤ MaxBudget_ch   （渠道上下限）
```

梯度通过有限差分数值计算，步长自适应于总预算规模。

## 快速开始

```bash
go build -o bin/mmm ./cmd/mmm
./bin/mmm -port 8082

# 训练 MMM 模型
curl -X POST http://localhost:8082/v1/fit \
  -d '{"channels":["sem","social","display"],"weeks":52}'

# 预算优化
curl -X POST http://localhost:8082/v1/optimize \
  -d '{"total_budget":1000000,"horizon_weeks":4}'
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/fit` | 训练 MMM 模型 |
| POST | `/v1/optimize` | 最优预算分配 |
| GET  | `/v1/contribution` | 各渠道贡献分解 |
| GET  | `/metrics` | Prometheus 指标 |

## 贡献分解输出示例

```json
{
  "baseline": 0.35,
  "channels": {
    "sem":     { "contribution": 0.28, "roi": 3.2 },
    "social":  { "contribution": 0.22, "roi": 2.1 },
    "display": { "contribution": 0.15, "roi": 1.8 }
  },
  "validation_mape": 0.11
}
```

## 模型评估指标

| 指标 | 说明 |
|------|------|
| MAPE | 验证集平均绝对百分比误差，目标 < 20% |
| R² | 决定系数 |
| ROI per channel | 每渠道边际 ROI |

## 技术栈

- **语言**: Go
- **依赖**: 无外部 ML 库，纯 Go 实现（OLS + 梯度上升）
- **数据源**: ClickHouse（周度汇总花费/转化数据）

## 目录结构

```
adortb-mmm/
├── cmd/mmm/
├── internal/
│   ├── aggregator/       # 数据聚合（按渠道/周汇总）
│   ├── api/              # HTTP handler + mock
│   ├── model/
│   │   ├── adstock.go    # 几何/加权 adstock 变换
│   │   ├── saturation.go # Hill 饱和曲线
│   │   ├── regression.go # Ridge 回归
│   │   ├── fit.go        # 网格搜索 + 全量训练
│   │   └── predict.go    # 预测
│   ├── optimizer/        # 预算优化（投影梯度上升）
│   └── reporting/        # 贡献分解 + 建议报告
```
