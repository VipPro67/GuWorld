export type BuyOrderExecution = {
  orderId: string;
  quantity: number;
  guInstanceIds?: string[];
};

export function buildBuyOrderExecution(
  orderId: string,
  quantity: number,
  itemType: "Gu" | "Material" | "Recipe",
  selectedGuInstanceIds: string[],
): BuyOrderExecution {
  if (itemType !== "Gu") return { orderId, quantity };

  if (selectedGuInstanceIds.length !== quantity) {
    throw new Error(`Cần chọn đúng ${quantity} Cổ trùng để bán`);
  }

  return { orderId, quantity, guInstanceIds: selectedGuInstanceIds };
}
