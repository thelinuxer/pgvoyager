import { FullConfig } from '@playwright/test';

async function globalTeardown(config: FullConfig) {
  console.log('\n🧹 Global Teardown: Cleaning up...\n');

  // PostgreSQL test database is left intact for debugging
  // It will be recreated on the next test run

  // PgVoyager SQLite state is cleaned in global-setup
  // No additional cleanup needed here

  console.log('✅ Global teardown complete!\n');
}

export default globalTeardown;
