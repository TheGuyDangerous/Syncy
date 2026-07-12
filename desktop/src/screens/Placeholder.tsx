import { Screen } from "../components/Screen";
import { EmptyState } from "../components/EmptyState";
import type { IconName } from "../components/Icon";

export default function Placeholder({
  title,
  description,
  icon,
  emptyTitle,
  emptyHint,
}: {
  title: string;
  description: string;
  icon: IconName;
  emptyTitle: string;
  emptyHint: string;
}) {
  return (
    <Screen title={title}>
      <p className="screen-desc">{description}</p>
      <section className="card">
        <EmptyState icon={icon} title={emptyTitle} hint={emptyHint} />
      </section>
    </Screen>
  );
}
