# ARCHITECTURE

## 项目是啥
终端贪吃蛇，Go 写的，UI 用 tcell 库直接画字符。代码两个文件：`main.go` 管游戏状态、输入、碰撞、combo；`renderer.go` 封装所有画图和样式。典型的「状态更新 + 渲染」分层。

## 核心数据结构
- **Point**：X/Y 坐标，全游戏通用的格子定位
- **Game**：主结构体
  - 蛇：`snake`（切片，头在尾部）、`dir`/`nextDir`（防止 180° 反向撞死）
  - 地图：`foods`（3 红食物）、`goldFood`（1 金食物）、`obstacles`（8 障碍）、`powerUp`（当前道具指针）
  - 状态：`score`/`highScore`、`boostEnd`/`slowEnd`（加速/减速截止时间）、`wallPass`（穿墙 flag）、`paused`/`over`/`startup`
  - Combo：`currentCombo`（1-5 倍率）、`lastEatTime`、`comboFlashEnd`
- **PowerType + PowerUp**：三个道具枚举 + 位置/类型/存在时长

## 主循环
初始化 tcell 后死循环：`Show()` 刷上一帧 → `PollEvent()` 等事件。输入改 `nextDir`（不能 180° 反向），过了 `moveInterval` 就调 `move()`（碰撞 + 吃食物 + 吃道具），每 12 秒调 `spawnPowerUp()`，最后 `renderer.Draw()` 画出来。帧间隔由 `currentInterval()` 根据 buff 动态算。

## 食物、障碍、Combo
开局障碍和食物都用 `randEmpty()` 随机撒 18×18 内格里，occupied map 保证不重叠。红：+10×combo、蛇+1 节；金：+30×combo，`boostEnd` 设 5 秒后。Combo 2 秒内连吃就 +1（顶 x5），超时从 x1 重来，每次吃食物蛇头按倍率颜色闪 300ms。

## 道具系统
每 12 秒刷一个，三种按生成顺序循环（`powerCycle % 3`），4 秒不吃消失。减速 8 秒移速×1.5；穿墙下次撞墙从对侧穿出（一次性）；缩短砍 3 节蛇尾（保底 1 节）。全是改截止时间/flag，主循环不跑独立 timer。

## 存档
最高分存 `~/.snake_score`。启动时 `os.UserHomeDir()` 拼路径，读不到从 0 开始。死亡破纪录或 q 退出前覆盖写回。
