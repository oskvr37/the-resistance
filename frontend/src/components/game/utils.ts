export const getRequiredTeamSize = (
  playerCount: number = 5,
  round: number = 1,
): number => {
  const matrix: Record<number, number[]> = {
    5: [2, 3, 2, 3, 3],
    6: [2, 3, 4, 3, 4],
    7: [2, 3, 3, 4, 4],
    8: [3, 4, 4, 5, 5],
    9: [3, 4, 4, 5, 5],
    10: [3, 4, 4, 5, 5],
  };
  // round - 1 because rounds are 1-indexed but arrays are 0-indexed
  return matrix[playerCount]?.[round - 1] || 2;
};
