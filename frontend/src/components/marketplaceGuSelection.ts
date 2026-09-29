export type BuyOrderExecution = {
  orderId: string;
  quantity: number;
  guInstanceIds?: string[];
};

export function reconcileGuSelection(
  eligibleGuInstanceIds: string[],
  selectedGuInstanceIds: string[],
  quantity: number,
): string[] {
  const desired = Math.max(0, Math.min(Math.floor(quantity), eligibleGuInstanceIds.length));
  const eligible = new Set(eligibleGuInstanceIds);
  const kept = selectedGuInstanceIds.filter((id, index) =>
    eligible.has(id) && selectedGuInstanceIds.indexOf(id) === index
  ).slice(0, desired);
  if (kept.length === desired) return kept;

  const selected = new Set(kept);
  for (const id of eligibleGuInstanceIds) {
    if (!selected.has(id)) {
      kept.push(id);
      selected.add(id);
      if (kept.length === desired) break;
    }
  }
  return kept;
}

export function toggleGuSelection(selectedGuInstanceIds: string[], instanceId: string): string[] {
  return selectedGuInstanceIds.includes(instanceId)
    ? selectedGuInstanceIds.filter((id) => id !== instanceId)
    : [...selectedGuInstanceIds, instanceId];
}

export function buildBuyOrderExecution(
  orderId: string,
  quantity: number,
  itemType: "Gu" | "Material" | "Recipe",
  selectedGuInstanceIds: string[],
  orderType: "BUY" | "SELL" = "BUY",
): BuyOrderExecution {
  if (itemType !== "Gu" || orderType === "SELL") return { orderId, quantity };

  if (selectedGuInstanceIds.length !== quantity) {
    throw new Error(`Cần chọn đúng ${quantity} Cổ trùng để bán`);
  }

  return { orderId, quantity, guInstanceIds: selectedGuInstanceIds };
}
