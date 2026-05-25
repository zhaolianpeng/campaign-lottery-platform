import { randomBytes, randomInt } from 'node:crypto';

export interface PrizeWeight {
  readonly id: string;
  readonly weight: number;
  readonly level: string;
}

export interface PityEngineConfig {
  readonly enabled: boolean;
  readonly softPityN: number;
  readonly pityFactor: number;
  readonly hardPityN: number;
  readonly targetPrizeId: string;
  readonly targetWeight: number;
}

export interface PityState {
  readonly userId: string;
  readonly campaignId: string;
  readonly consecutiveMisses: number;
  readonly pityMultiplier: number;
  readonly hasUPPoolGuarantee: boolean;
}

export interface ProbabilityDrawResult {
  readonly prizeId: string;
  readonly prizeLevel: string;
  readonly consecutiveMisses: number;
  readonly pityMultiplier: number;
  readonly isHardPity: boolean;
  readonly isUPPoolWin: boolean;
}

export class AliasTable {
  private readonly probabilities: readonly number[];
  private readonly aliases: readonly number[];

  private constructor(probabilities: readonly number[], aliases: readonly number[]) {
    this.probabilities = probabilities;
    this.aliases = aliases;
  }

  public static fromWeights(weights: readonly number[]): AliasTable | null {
    if (weights.length === 0) {
      return null;
    }

    const total = weights.reduce((sum, weight) => sum + Math.max(0, weight), 0);
    if (total <= 0) {
      return null;
    }

    const size = weights.length;
    const scaled = weights.map((weight) => (size * Math.max(0, weight)) / total);
    const probabilities = [...scaled];
    const aliases = Array.from({ length: size }, () => -1);
    const small: number[] = [];
    const large: number[] = [];

    scaled.forEach((value, index) => {
      if (value < 1) {
        small.push(index);
      } else {
        large.push(index);
      }
    });

    while (small.length > 0 && large.length > 0) {
      const low = small.pop();
      const high = large.pop();
      if (low === undefined || high === undefined) {
        break;
      }

      probabilities[low] = scaled[low];
      aliases[low] = high;
      scaled[high] -= 1 - scaled[low];

      if (scaled[high] < 1) {
        small.push(high);
      } else {
        large.push(high);
      }
    }

    for (const index of [...large, ...small]) {
      probabilities[index] = 1;
    }

    return new AliasTable(probabilities, aliases);
  }

  public next(): number {
    const index = randomInt(this.probabilities.length);
    const threshold = randomBytes(4).readUInt32BE(0) / 0xffffffff;
    return threshold < this.probabilities[index] ? index : this.aliases[index];
  }
}

export class PityTracker {
  private readonly states = new Map<string, PityState>();

  public get(userId: string, campaignId: string): PityState {
    return (
      this.states.get(this.key(userId, campaignId)) ?? {
        userId,
        campaignId,
        consecutiveMisses: 0,
        pityMultiplier: 1,
        hasUPPoolGuarantee: false,
      }
    );
  }

  public incrementMiss(userId: string, campaignId: string): PityState {
    const current = this.get(userId, campaignId);
    const nextState: PityState = {
      ...current,
      consecutiveMisses: current.consecutiveMisses + 1,
      pityMultiplier: 1 + 1 / (current.consecutiveMisses + 2),
    };
    this.states.set(this.key(userId, campaignId), nextState);
    return nextState;
  }

  public reset(userId: string, campaignId: string): void {
    const current = this.get(userId, campaignId);
    this.states.set(this.key(userId, campaignId), {
      userId,
      campaignId,
      consecutiveMisses: 0,
      pityMultiplier: 1,
      hasUPPoolGuarantee: current.hasUPPoolGuarantee,
    });
  }

  public setUPPoolGuarantee(userId: string, campaignId: string, guaranteed: boolean): void {
    const current = this.get(userId, campaignId);
    this.states.set(this.key(userId, campaignId), {
      ...current,
      hasUPPoolGuarantee: guaranteed,
    });
  }

  private key(userId: string, campaignId: string): string {
    return `${userId}:${campaignId}`;
  }
}

export function calculateEffectiveWeight(
  baseWeight: number,
  config: PityEngineConfig,
  consecutiveMisses: number,
): number {
  if (!config.enabled || config.softPityN <= 0 || consecutiveMisses < config.softPityN) {
    return baseWeight;
  }

  if (config.hardPityN > 0 && consecutiveMisses >= config.hardPityN) {
    return Number.MAX_SAFE_INTEGER;
  }

  const missesPastThreshold = consecutiveMisses - config.softPityN + 1;
  return baseWeight * (1 + config.pityFactor * missesPastThreshold);
}

export class ProbabilityEngine {
  private readonly weights = new Map<string, number>();
  private readonly levels = new Map<string, string>();
  private readonly prizeIds: readonly string[];

  public constructor(
    private readonly missWeight: number,
    prizeWeights: readonly PrizeWeight[],
  ) {
    for (const prizeWeight of prizeWeights) {
      this.weights.set(prizeWeight.id, prizeWeight.weight);
      this.levels.set(prizeWeight.id, prizeWeight.level);
    }
    this.prizeIds = prizeWeights.map((item) => item.id).sort();
  }

  public draw(
    config: PityEngineConfig,
    tracker: PityTracker,
    userId: string,
    campaignId: string,
  ): ProbabilityDrawResult {
    const state = tracker.get(userId, campaignId);
    const dynamicWeights = new Map(this.weights);
    let pityMultiplier = 1;

    if (config.enabled && state.consecutiveMisses >= config.softPityN && config.targetWeight > 0) {
      const effectiveWeight = calculateEffectiveWeight(
        config.targetWeight,
        config,
        state.consecutiveMisses,
      );
      dynamicWeights.set(config.targetPrizeId, effectiveWeight);
      pityMultiplier = effectiveWeight / config.targetWeight;

      if (config.hardPityN > 0 && state.consecutiveMisses >= config.hardPityN) {
        return {
          prizeId: config.targetPrizeId,
          prizeLevel: this.levels.get(config.targetPrizeId) ?? '',
          consecutiveMisses: state.consecutiveMisses,
          pityMultiplier: Number.MAX_SAFE_INTEGER,
          isHardPity: true,
          isUPPoolWin: false,
        };
      }
    }

    const samplerIds = this.missWeight > 0 ? [''] : [];
    const samplerWeights = this.missWeight > 0 ? [this.missWeight] : [];
    for (const prizeId of this.prizeIds) {
      samplerIds.push(prizeId);
      samplerWeights.push(dynamicWeights.get(prizeId) ?? 0);
    }

    const alias = AliasTable.fromWeights(samplerWeights);
    const selectedId = alias ? samplerIds[alias.next()] ?? '' : '';

    if (selectedId === '') {
      tracker.incrementMiss(userId, campaignId);
      return {
        prizeId: '',
        prizeLevel: '',
        consecutiveMisses: state.consecutiveMisses,
        pityMultiplier,
        isHardPity: false,
        isUPPoolWin: false,
      };
    }

    tracker.reset(userId, campaignId);
    return {
      prizeId: selectedId,
      prizeLevel: this.levels.get(selectedId) ?? '',
      consecutiveMisses: state.consecutiveMisses,
      pityMultiplier,
      isHardPity: false,
      isUPPoolWin: false,
    };
  }

  public drawMultiple(
    count: number,
    config: PityEngineConfig,
    tracker: PityTracker,
    userId: string,
    campaignId: string,
  ): readonly ProbabilityDrawResult[] {
    return Array.from({ length: count }, () => this.draw(config, tracker, userId, campaignId));
  }
}
