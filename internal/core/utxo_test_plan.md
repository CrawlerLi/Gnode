# `internal/core/utxo.go` 单元测试文档（基于当前版本）

## 1. 目标

针对当前 `UTXOSet` 的 3 个方法做行为验证：

- `UpdateUTXO(b *Block, dbTx database.Tx) error`
- `FindSpendableUTXOS(amount int, pubkeyHash []byte) (map[string][]int, int, error)`
- `FindTransaction(txID []byte, outindex int) (TxOutput, error)`

重点验证：

- 正常路径结果正确。
- bucket 缺失、数据缺失、反序列化失败时错误路径可预期。
- 返回值与错误值组合符合当前函数签名。

## 2. 当前实现注意点（影响测试设计）

1. `UpdateUTXO` 写入 value 使用 `tx.SerializeTxOutput()`。  
2. `FindSpendableUTXOS` 与 `FindTransaction` 都依赖 bucket 名称 `UTXOSet`。  
3. `FindSpendableUTXOS` 在 `acc >= amount` 时提前 `break`。  
4. `FindTransaction` 对找不到 key 的情况返回 `"tx output does not exist"` 错误。  

## 3. 测试隔离策略

- 每个测试创建独立临时 DB 文件（例如 `test_utxo_<case>.db`）。
- 每个测试自行决定是否创建 `UTXOSet` bucket，用于覆盖正常/异常路径。
- 不依赖全局链状态，不调用区块链主流程。

## 4. 用例清单

### 4.1 `FindSpendableUTXOS` 正常命中

- 前置：
  - 创建 `UTXOSet` bucket。
  - 手动写入多条 UTXO 记录（至少两条 `ScriptPubkey` 匹配目标地址，一条不匹配）。
- 行为：调用 `FindSpendableUTXOS(amount=目标值, pubkeyHash=目标)`。
- 断言：
  - `err == nil`
  - `acc >= amount`
  - 返回 map 中只包含匹配 `pubkeyHash` 的输出索引

### 4.2 `FindSpendableUTXOS` 金额不足

- 前置：写入总额小于 `amount` 的匹配 UTXO。
- 行为：查询更大金额。
- 断言：
  - `err == nil`
  - `acc < amount`
  - 返回 map 非空且内容与实际可用 UTXO 一致

### 4.3 `FindSpendableUTXOS` bucket 缺失

- 前置：不创建 `UTXOSet` bucket。
- 行为：直接调用查询。
- 断言：
  - `err != nil`
  - `payable == nil`
  - `acc == 0`

### 4.4 `FindSpendableUTXOS` 反序列化失败

- 前置：在 `UTXOSet` 写入非法 value（非 `TxOutput` gob 数据）。
- 行为：调用查询。
- 断言：
  - `err != nil`
  - 返回值为失败分支（当前实现是 `nil, 0, err`）

### 4.5 `FindTransaction` 正常命中

- 前置：写入 `<txid>:<outindex>` 对应的合法 `TxOutput`。
- 行为：调用 `FindTransaction(txid, outindex)`。
- 断言：
  - `err == nil`
  - `TxOutput.Value`、`ScriptPubkey` 与写入一致

### 4.6 `FindTransaction` key 不存在

- 前置：bucket 存在，但目标 key 不存在。
- 行为：查询不存在 key。
- 断言：
  - `err != nil`
  - 返回 `TxOutput` 为零值

### 4.7 `FindTransaction` bucket 缺失

- 前置：不创建 `UTXOSet` bucket。
- 行为：调用查询。
- 断言：
  - `err != nil`
  - 返回 `TxOutput` 为零值

### 4.8 `FindTransaction` 反序列化失败

- 前置：key 存在，但 value 非法。
- 行为：查询该 key。
- 断言：
  - `err != nil`
  - 返回 `TxOutput` 为零值

### 4.9 `UpdateUTXO` 删除已花费输出

- 前置：
  - 在 `UTXOSet` 预写一个旧输出 `prevTxID:0`。
  - 构造一个非 coinbase 交易，输入引用 `prevTxID:0`。
  - 构造 block 仅包含该交易。
- 行为：在 `db.Update` 事务中调用 `UpdateUTXO`。
- 断言：
  - `err == nil`
  - `prevTxID:0` 被删除

### 4.10 `UpdateUTXO` bucket 缺失

- 前置：不创建 `UTXOSet` bucket，构造最小 block。
- 行为：在事务中调用 `UpdateUTXO`。
- 断言：
  - `err != nil`（来自 `bucket.Put` / `bucket.Delete`）

## 5. 推荐测试文件结构

文件：`internal/core/utxo_test.go`

建议函数：

- `TestUTXOSet_FindSpendableUTXOS_OK`
- `TestUTXOSet_FindSpendableUTXOS_Insufficient`
- `TestUTXOSet_FindSpendableUTXOS_BucketNotFound`
- `TestUTXOSet_FindSpendableUTXOS_BadValue`
- `TestUTXOSet_FindTransaction_OK`
- `TestUTXOSet_FindTransaction_NotFound`
- `TestUTXOSet_FindTransaction_BucketNotFound`
- `TestUTXOSet_FindTransaction_BadValue`
- `TestUTXOSet_UpdateUTXO_DeleteSpent`
- `TestUTXOSet_UpdateUTXO_BucketNotFound`

## 6. 测试辅助函数建议

1. `newTestDB(t *testing.T, createBucket bool) (database.DB, cleanup func())`  
2. `putRawUTXO(t *testing.T, db database.DB, key string, value []byte)`  
3. `mustSerializeTxOutput(t *testing.T, out TxOutput) []byte`  
说明：不要复用 `SerializeTxOutput`，因为它当前序列化的是 `Transaction`。建议在测试里用 `gob` 单独序列化 `TxOutput`，确保可控。

## 7. 断言规范

- map 断言：
  - 先判 key 是否存在，再比索引切片长度和值。
- 字节比较：
  - 使用 `bytes.Equal`。
- 错误断言：
  - 至少判断 `err != nil` / `err == nil`。
  - 如果断言错误文案，使用 `strings.Contains`，避免过度脆弱。

## 8. 执行命令

仅跑 UTXO 相关测试：

```bash
go test ./internal/core -run UTXO -v
```

如果当前 `internal/core` 有其它未完成代码导致编译失败，可先让 UTXO 测试落在可独立编译的包后再执行。

