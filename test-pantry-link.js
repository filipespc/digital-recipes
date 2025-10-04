const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('Testing pantry item linking functionality...\n');

    // Navigate to the recipe edit page
    await page.goto('http://localhost:3000/recipes/3/edit');
    await page.waitForLoadState('networkidle');
    console.log('✅ Page loaded successfully');

    // Check for console errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Wait for ingredients to load
    await page.waitForSelector('.space-y-3', { timeout: 5000 });
    console.log('✅ Ingredients section loaded');

    // Find the first ingredient's dropdown button
    const firstDropdown = await page.locator('.border-green-200').first().locator('button').first();

    if (await firstDropdown.count() > 0) {
      console.log('✅ Found linked ingredient with dropdown');

      // Click to open dropdown
      await firstDropdown.click();
      await page.waitForTimeout(500);

      // Check if dropdown opened
      const dropdownMenu = page.locator('.absolute.z-20.mt-1.w-full.bg-white.shadow-lg');
      if (await dropdownMenu.count() > 0) {
        console.log('✅ Dropdown menu opened');

        // Wait for pantry items to load
        await page.waitForTimeout(1000);

        // Try to select a different pantry item
        const pantryItems = await page.locator('button:has-text("Linked to Pantry")').all();
        if (pantryItems.length > 0) {
          // Click on a pantry item
          const itemToClick = await page.locator('.absolute.z-20 button').nth(1);
          const itemText = await itemToClick.textContent();
          console.log(`📝 Selecting pantry item: ${itemText}`);

          await itemToClick.click();
          await page.waitForTimeout(1000);

          // Check if the UI updated
          const updatedDropdownText = await firstDropdown.textContent();
          console.log(`✅ Dropdown now shows: ${updatedDropdownText}`);

          if (updatedDropdownText && updatedDropdownText.trim() !== 'Select pantry item...') {
            console.log('✅ UI updated successfully after selection!');
          } else {
            console.log('❌ UI did not update properly');
          }
        }
      }
    } else {
      // Try the "Create new instead" flow
      console.log('📝 Testing "Create new instead" button...');
      const createNewButton = await page.locator('button:has-text("Create new instead")').first();

      if (await createNewButton.count() > 0) {
        await createNewButton.click();
        await page.waitForTimeout(500);

        // Check if switched to "new" state
        const linkExistingButton = await page.locator('button:has-text("Link to existing instead")').first();
        if (await linkExistingButton.count() > 0) {
          console.log('✅ Successfully switched to "new" state');

          // Switch back to test linking
          await linkExistingButton.click();
          await page.waitForTimeout(500);

          // Now test the dropdown
          const dropdown = await page.locator('.border-green-200').first().locator('button').first();
          await dropdown.click();
          await page.waitForTimeout(1000);

          console.log('✅ Dropdown opened after switching back');
        }
      }
    }

    // Check for React key prop errors
    if (consoleErrors.length > 0) {
      const keyErrors = consoleErrors.filter(err => err.includes('unique "key" prop'));
      if (keyErrors.length > 0) {
        console.log('\n❌ React key prop errors detected:');
        keyErrors.forEach(err => console.log(`  - ${err}`));
      } else {
        console.log('\n✅ No React key prop errors detected!');
      }
    } else {
      console.log('\n✅ No console errors detected!');
    }

    // Take a screenshot
    await page.screenshot({
      path: '/home/filipe-carneiro/projects/digital-recipes/.playwright-mcp/pantry-link-test.png',
      fullPage: true
    });
    console.log('\n📸 Screenshot saved to .playwright-mcp/pantry-link-test.png');

    console.log('\n✅ Test completed successfully!');

  } catch (error) {
    console.error('❌ Test failed:', error.message);
    await page.screenshot({
      path: '/home/filipe-carneiro/projects/digital-recipes/.playwright-mcp/pantry-link-error.png',
      fullPage: true
    });
  } finally {
    await browser.close();
  }
})();