from __future__ import annotations

from dataclasses import dataclass

import gymnasium as gym
import numpy as np
import pandas as pd

from features import FEATURE_COLS, add_features


@dataclass
class Costs:
    fee_bps: float = 10.0       # 0.10%
    slippage_bps: float = 5.0   # 0.05%


class TradingEnv(gym.Env):
    """
    Минимальная среда:
    - Действия: 0 hold, 1 buy (войти/увеличить), 2 sell (выйти/уменьшить)
    - Состояние: фичи по свечам + текущая позиция (0/1)
    - Награда = рост equity (в плюс к балансу) минус costs и риск-штраф.
      То есть цель агента: увеличивать баланс после всех расходов.
    """

    metadata = {"render_modes": []}

    def __init__(self, df: pd.DataFrame, costs: Costs | None = None):
        super().__init__()
        self.df = add_features(df).reset_index(drop=True)
        self.costs = costs or Costs()

        self.action_space = gym.spaces.Discrete(3)
        self.observation_space = gym.spaces.Box(
            low=-10.0, high=10.0, shape=(len(FEATURE_COLS) + 1,), dtype=np.float32
        )

        self.i = 0
        self.pos = 0  # 0 cash, 1 long
        self.entry_price = 0.0
        self.equity = 1.0
        self.peak = 1.0

    def reset(self, *, seed=None, options=None):
        super().reset(seed=seed)
        self.i = 0
        self.pos = 0
        self.entry_price = 0.0
        self.equity = 1.0
        self.peak = 1.0
        return self._obs(), {}

    def step(self, action: int):
        done = False
        info = {}

        price = float(self.df.loc[self.i, "close"])
        next_i = self.i + 1
        if next_i >= len(self.df):
            done = True
            return self._obs(), 0.0, done, False, info

        # costs
        fee = self.costs.fee_bps / 10_000.0
        slip = self.costs.slippage_bps / 10_000.0

        # execute action at current price (simplified)
        if action == 1 and self.pos == 0:
            self.pos = 1
            self.entry_price = price * (1 + slip)
            self.equity *= (1 - fee)
        elif action == 2 and self.pos == 1:
            exit_price = price * (1 - slip)
            pnl = (exit_price / self.entry_price) - 1.0
            self.equity *= (1 + pnl) * (1 - fee)
            self.pos = 0
            self.entry_price = 0.0

        # mark-to-market one step forward
        next_price = float(self.df.loc[next_i, "close"])
        prev_equity = self.equity
        if self.pos == 1 and self.entry_price > 0:
            # unrealized pnl affects equity (paper, simplified)
            self.equity = prev_equity * (next_price / price)

        self.peak = max(self.peak, self.equity)
        drawdown = (self.peak - self.equity) / max(self.peak, 1e-9)

        # Положительная награда только при реальном росте equity.
        reward = float(np.log(max(self.equity, 1e-9)) - np.log(max(prev_equity, 1e-9)))
        reward -= 0.10 * float(drawdown)  # penalty

        self.i = next_i
        return self._obs(), reward, done, False, info

    def _obs(self) -> np.ndarray:
        row = self.df.loc[self.i, FEATURE_COLS].astype(float).to_numpy()
        pos = np.array([float(self.pos)], dtype=float)
        obs = np.concatenate([row, pos], axis=0).astype(np.float32)
        # clip for stability
        return np.clip(obs, -10.0, 10.0)

