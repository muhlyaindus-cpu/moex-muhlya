from __future__ import annotations

import sys
from pathlib import Path

_LAB = Path(__file__).resolve().parent
if str(_LAB) not in sys.path:
    sys.path.insert(0, str(_LAB))

import argparse
import os

import pandas as pd
from stable_baselines3 import PPO

from env import TradingEnv


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", required=True)
    ap.add_argument("--model-out", required=True)
    ap.add_argument("--timesteps", type=int, default=50_000)
    args = ap.parse_args()

    df = pd.read_csv(args.data)
    if "close" not in df.columns:
        raise SystemExit("CSV must contain 'close' column")

    env = TradingEnv(df)
    model = PPO("MlpPolicy", env, verbose=1, n_steps=1024, batch_size=256)
    model.learn(total_timesteps=args.timesteps)

    os.makedirs(os.path.dirname(args.model_out), exist_ok=True)
    model.save(args.model_out)
    print(f"Saved model: {args.model_out}")


if __name__ == "__main__":
    main()

