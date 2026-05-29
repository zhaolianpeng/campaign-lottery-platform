'use client';

import { InventoryTabPanel } from '../components/inventory-tab-panel';
import type { ComponentProps } from 'react';

export type InventoryTabProps = ComponentProps<typeof InventoryTabPanel>;

export function InventoryTab(props: InventoryTabProps): React.ReactNode {
  return <InventoryTabPanel {...props} />;
}
