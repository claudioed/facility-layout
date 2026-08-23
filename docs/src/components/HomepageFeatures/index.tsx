import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  to: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Coded location hierarchy',
    to: '/docs/business-context/location-code',
    description: (
      <>
        The industry-standard <code>Site-Area-Zone-Aisle-Bay-Level-Position</code>{' '}
        address, modelled as a value object with seven typed segments — never a
        free-text string, never an opaque id.
      </>
    ),
  },
  {
    title: 'Chain of custody',
    to: '/docs/ddd/invariants',
    description: (
      <>
        Registering a slot resolves its Site → Zone → Aisle chain and rejects
        anything that does not land on existing, <em>Active</em> structure. No
        orphan locations, ever.
      </>
    ),
  },
  {
    title: 'Placement rules',
    to: '/docs/adr/0003-placement-rules-at-registration-time',
    description: (
      <>
        The guard that keeps ambient shelving out of the frozen zone — enforced
        once, at registration time, naming the exact rule it refused on. Every
        stored slot is legal by construction.
      </>
    ),
  },
  {
    title: 'Draw the warehouse',
    to: '/docs/api-reference/drawing-the-warehouse',
    description: (
      <>
        Pre-ordered nested layout, an index-aligned 2D zone grid, and a
        server-rendered SVG floor plan. Renderable structure, not rows a client
        has to join and sort.
      </>
    ),
  },
];

function Feature({title, to, description}: FeatureItem) {
  return (
    <div className={clsx('col col--3')}>
      <div className={styles.featureCard}>
        <Heading as="h3" className={styles.featureTitle}>
          <Link to={to}>{title}</Link>
        </Heading>
        <p className={styles.featureBody}>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
