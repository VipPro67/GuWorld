import React from "react";

export interface WikiPage {
  id: string;
  title: string;
  category: string;
  subCategory?: string;
  keywords: string[];
  content: React.ReactNode;
  cards?: { title: string; targetId: string; description: string }[];
}
