# Changelog

## v0.1.0
_发布日期：2026-11-18_

- feat: 增加 Extra/Until 字段与跨 key 原子批量接口

## v0.0.3
_发布日期：2026-10-12_

- 更新 go-storage
- fix(connection-manager): 漏洞4 + 文档化抽象语义边界
- docs: 明确 CapabilityCAS 必须是线性一致，堵住"最终一致条件写"误声明陷阱
- 修复理论漏洞：TimeProvider 解绑 + 删除可降级为墓碑
- feat: 编码分布式锁两个必要条件(CAS+可靠时间源)到 Storage 接口

## v0.0.2
_发布日期：2023-08-07_

- 修改接口方法名称

## v0.0.1
_发布日期：2023-08-07_

- 完善文档
