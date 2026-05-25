import { AliasTable, calculateEffectiveWeight, PityTracker, ProbabilityEngine } from './probability';

describe('probability engine', () => {
  it('builds an alias table for weighted samples', () => {
    const table = AliasTable.fromWeights([1, 2, 3]);

    expect(table).not.toBeNull();
    expect(table?.next()).toBeGreaterThanOrEqual(0);
  });

  it('increases target weight after soft pity threshold', () => {
    const actual = calculateEffectiveWeight(
      2,
      {
        enabled: true,
        softPityN: 3,
        pityFactor: 0.5,
        hardPityN: 10,
        targetPrizeId: 'secret',
        targetWeight: 2,
      },
      4,
    );

    expect(actual).toBe(4);
  });

  it('returns the target prize when hard pity is reached', () => {
    const tracker = new PityTracker();
    tracker.incrementMiss('user-1', 'campaign-1');
    tracker.incrementMiss('user-1', 'campaign-1');
    const engine = new ProbabilityEngine(100, [
      { id: 'secret', weight: 1, level: 'secret' },
      { id: 'common', weight: 99, level: 'common' },
    ]);

    const result = engine.draw(
      {
        enabled: true,
        softPityN: 1,
        pityFactor: 0.1,
        hardPityN: 2,
        targetPrizeId: 'secret',
        targetWeight: 1,
      },
      tracker,
      'user-1',
      'campaign-1',
    );

    expect(result.prizeId).toBe('secret');
    expect(result.isHardPity).toBe(true);
  });
});
