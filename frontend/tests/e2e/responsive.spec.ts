import { test, expect, devices } from '@playwright/test';

test.describe('Responsive Design E2E Tests', () => {
  // Modern device configurations for 2025
  const deviceConfigs = [
    { name: 'iPhone 15 Pro', width: 393, height: 852, isMobile: true, hasTouch: true },
    { name: 'iPhone 15 Pro Landscape', width: 852, height: 393, isMobile: true, hasTouch: true },
    { name: 'Samsung Galaxy S24', width: 384, height: 854, isMobile: true, hasTouch: true },
    { name: 'iPad Air', width: 820, height: 1180, isTablet: true, hasTouch: true },
    { name: 'iPad Air Landscape', width: 1180, height: 820, isTablet: true, hasTouch: true },
    { name: 'MacBook Pro 14"', width: 1512, height: 982, isDesktop: true, hasTouch: false },
    { name: 'Desktop 4K', width: 1920, height: 1080, isDesktop: true, hasTouch: false },
    { name: 'Ultrawide Monitor', width: 2560, height: 1440, isDesktop: true, hasTouch: false }
  ];

  test.beforeEach(async ({ page }) => {
    // Mock API response with sufficient images for testing
    await page.route('**/api/images/random', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '1',
            filename: 'image1.jpg',
            url: '/api/images/1',
            altText: 'Beautiful mountain landscape with sunrise colors reflecting on pristine lake',
            title: 'Mountain Sunrise',
            tags: ['nature', 'motivational'],
            weight: 5
          },
          {
            id: '2',
            filename: 'image2.jpg',
            url: '/api/images/2',
            altText: 'Peaceful forest path winding through ancient tall trees with dappled sunlight',
            title: 'Forest Path',
            tags: ['nature', 'peaceful'],
            weight: 3
          },
          {
            id: '3',
            filename: 'image3.jpg',
            url: '/api/images/3',
            altText: 'Powerful ocean waves crashing against rugged rocky cliffs at sunset',
            title: 'Ocean Power',
            tags: ['ocean', 'strength'],
            weight: 4
          },
          {
            id: '4',
            filename: 'image4.jpg',
            url: '/api/images/4',
            altText: 'Vibrant sunset sky with dramatic clouds over rolling green hills',
            title: 'Sunset Hills',
            tags: ['sunset', 'inspirational'],
            weight: 5
          },
          {
            id: '5',
            filename: 'image5.jpg',
            url: '/api/images/5',
            altText: 'Majestic waterfall cascading down moss-covered rocks in lush forest',
            title: 'Waterfall Majesty',
            tags: ['water', 'nature'],
            weight: 4
          }
        ])
      });
    });

    // Mock image responses
    await page.route('**/api/images/*', route => {
      const base64Image = 'data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAhEAACAQMDBQAAAAAAAAAAAAABAgMABAUGIWGRkqGxwf/EABUBAQEAAAAAAAAAAAAAAAAAAAMF/8QAGhEAAgIDAAAAAAAAAAAAAAAAAAECEgMRkf/aAAwDAQACEQMRAD8AltJagyeH0AthI5xdrLcNM91BF5pX2HaH9bcfaSXWGaRmknyLDSjlcupqmwJbklk5xGxl4g==';
      route.fulfill({
        status: 200,
        contentType: 'image/jpeg',
        body: Buffer.from(base64Image.split(',')[1], 'base64')
      });
    });
  });

  deviceConfigs.forEach(device => {
    test(`Responsive layout works on ${device.name}`, async ({ page }) => {
      // Set viewport to device specifications
      await page.setViewportSize({ width: device.width, height: device.height });

      // Simulate touch capability for mobile/tablet devices
      if (device.hasTouch) {
        await page.addInitScript(() => {
          Object.defineProperty(navigator, 'maxTouchPoints', {
            writable: false,
            value: 5
          });
        });
      }

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Verify basic page structure
      const main = page.locator('main');
      await expect(main).toBeVisible();

      // Check image grid responsiveness
      const imageContainers = page.locator('[data-testid="image-container"]');
      const imageCount = await imageContainers.count();
      expect(imageCount).toBeGreaterThanOrEqual(3);
      expect(imageCount).toBeLessThanOrEqual(5);

      // Verify all images are visible and properly sized
      for (let i = 0; i < imageCount; i++) {
        const container = imageContainers.nth(i);
        const image = container.locator('img');

        await expect(image).toBeVisible();

        const box = await image.boundingBox();
        expect(box?.width).toBeGreaterThan(50);
        expect(box?.height).toBeGreaterThan(50);

        // Images should not exceed viewport width (with margin)
        expect(box?.width).toBeLessThanOrEqual(device.width - 20);
      }

      // Test layout behavior based on device type
      if (device.isMobile) {
        await testMobileLayout(page, device);
      } else if (device.isTablet) {
        await testTabletLayout(page, device);
      } else if (device.isDesktop) {
        await testDesktopLayout(page, device);
      }

      // Verify no horizontal scrolling
      const hasHorizontalScroll = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });
      expect(hasHorizontalScroll).toBeFalsy();

      // Check navigation menu responsiveness
      await testNavigationResponsiveness(page, device);

      // Verify text readability
      await testTextReadability(page, device);

      // Test interactive element sizes for touch devices
      if (device.hasTouch) {
        await testTouchTargetSizes(page);
      }
    });
  });

  async function testMobileLayout(page, device) {
    const imageContainers = page.locator('[data-testid="image-container"]');
    const containerCount = await imageContainers.count();

    // Mobile should typically show 1-2 images per row
    if (containerCount >= 2) {
      const container1 = imageContainers.nth(0);
      const container2 = imageContainers.nth(1);

      const box1 = await container1.boundingBox();
      const box2 = await container2.boundingBox();

      // Check if images are stacked vertically or side-by-side
      const isStacked = box2!.y > box1!.y + (box1!.height / 2);
      const isSideBySide = Math.abs(box1!.y - box2!.y) < 50;

      expect(isStacked || isSideBySide).toBeTruthy();

      if (isSideBySide) {
        // If side-by-side, each should take roughly half the width
        const expectedMaxWidth = (device.width - 60) / 2; // Account for padding/gaps
        expect(box1!.width).toBeLessThanOrEqual(expectedMaxWidth + 20);
        expect(box2!.width).toBeLessThanOrEqual(expectedMaxWidth + 20);
      }
    }

    // Check that content doesn't require horizontal scrolling
    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
    expect(bodyWidth).toBeLessThanOrEqual(device.width + 10);
  }

  async function testTabletLayout(page, device) {
    const imageContainers = page.locator('[data-testid="image-container"]');
    const containerCount = await imageContainers.count();

    // Tablet should show 2-3 images per row efficiently
    if (containerCount >= 3) {
      const boxes = [];
      for (let i = 0; i < Math.min(3, containerCount); i++) {
        boxes.push(await imageContainers.nth(i).boundingBox());
      }

      // Check for reasonable distribution across width
      const usedWidth = Math.max(...boxes.map(box => box!.x + box!.width));
      const availableWidth = device.width - 40; // Account for margins

      // Should use at least 60% of available width for multiple images
      if (containerCount > 1) {
        expect(usedWidth).toBeGreaterThan(availableWidth * 0.6);
      }
    }
  }

  async function testDesktopLayout(page, device) {
    const imageContainers = page.locator('[data-testid="image-container"]');
    const containerCount = await imageContainers.count();

    // Desktop should efficiently use horizontal space
    if (containerCount >= 3) {
      const boxes = [];
      for (let i = 0; i < containerCount; i++) {
        boxes.push(await imageContainers.nth(i).boundingBox());
      }

      // Images should be distributed across the width
      const leftmost = Math.min(...boxes.map(box => box!.x));
      const rightmost = Math.max(...boxes.map(box => box!.x + box!.width));
      const usedWidth = rightmost - leftmost;

      // Should use substantial portion of screen width
      expect(usedWidth).toBeGreaterThan(device.width * 0.5);
    }

    // Desktop images can be larger but should maintain reasonable proportions
    const images = page.locator('[data-testid="image-container"] img');
    const firstImage = images.first();
    const box = await firstImage.boundingBox();

    // Images should be substantial size on desktop
    expect(box?.width).toBeGreaterThan(200);
    expect(box?.height).toBeGreaterThan(150);
  }

  async function testNavigationResponsiveness(page, device) {
    // Look for navigation elements
    const nav = page.locator('nav').or(
      page.locator('[data-testid="navigation"]').or(
        page.locator('header nav')
      )
    );

    if (await nav.isVisible()) {
      const navBox = await nav.boundingBox();
      expect(navBox?.width).toBeLessThanOrEqual(device.width);

      // On mobile, navigation might be collapsed into hamburger menu
      if (device.isMobile) {
        const hamburger = page.locator('[data-testid="menu-toggle"]').or(
          page.locator('.hamburger').or(
            page.locator('button:has-text("menu")')
          )
        );

        if (await hamburger.isVisible()) {
          const hamburgerBox = await hamburger.boundingBox();
          // Touch-friendly size for mobile
          expect(hamburgerBox?.width).toBeGreaterThanOrEqual(44);
          expect(hamburgerBox?.height).toBeGreaterThanOrEqual(44);
        }
      }
    }

    // Check for any buttons or interactive elements in header/nav
    const navButtons = page.locator('nav button, header button');
    const buttonCount = await navButtons.count();

    for (let i = 0; i < buttonCount; i++) {
      const button = navButtons.nth(i);
      if (await button.isVisible()) {
        const buttonBox = await button.boundingBox();

        if (device.hasTouch) {
          // Touch targets should be minimum 44px
          expect(buttonBox?.width).toBeGreaterThanOrEqual(44);
          expect(buttonBox?.height).toBeGreaterThanOrEqual(44);
        } else {
          // Desktop can have smaller targets
          expect(buttonBox?.width).toBeGreaterThanOrEqual(24);
          expect(buttonBox?.height).toBeGreaterThanOrEqual(24);
        }
      }
    }
  }

  async function testTextReadability(page, device) {
    // Check text elements for appropriate sizing
    const textElements = page.locator('h1, h2, h3, h4, h5, h6, p, span, div').filter({
      hasText: /\S+/ // Has non-whitespace content
    });

    const visibleTextCount = await textElements.count();
    let checkedElements = 0;

    for (let i = 0; i < Math.min(10, visibleTextCount); i++) {
      const element = textElements.nth(i);
      if (await element.isVisible() && checkedElements < 5) {
        const fontSize = await element.evaluate(el => {
          const computed = window.getComputedStyle(el);
          return parseInt(computed.fontSize);
        });

        // Font sizes should be readable on all devices
        if (device.isMobile) {
          expect(fontSize).toBeGreaterThanOrEqual(14); // Minimum mobile readability
          expect(fontSize).toBeLessThanOrEqual(28);
        } else {
          expect(fontSize).toBeGreaterThanOrEqual(12); // Desktop can be slightly smaller
          expect(fontSize).toBeLessThanOrEqual(36);
        }

        checkedElements++;
      }
    }
  }

  async function testTouchTargetSizes(page) {
    // Find all interactive elements
    const interactiveElements = page.locator('button, a, input, [role="button"], [tabindex="0"]');
    const elementCount = await interactiveElements.count();

    for (let i = 0; i < Math.min(10, elementCount); i++) {
      const element = interactiveElements.nth(i);
      if (await element.isVisible()) {
        const box = await element.boundingBox();

        // Touch targets should meet accessibility guidelines (44px minimum)
        expect(box?.width).toBeGreaterThanOrEqual(44);
        expect(box?.height).toBeGreaterThanOrEqual(44);
      }
    }
  }

  test('Orientation change handling', async ({ page }) => {
    // Start with mobile portrait
    await page.setViewportSize({ width: 393, height: 852 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Get initial layout information
    const images = page.locator('[data-testid="image-container"] img');
    const initialCount = await images.count();
    expect(initialCount).toBeGreaterThanOrEqual(3);

    // Get initial positions for comparison
    const initialBoxes = [];
    for (let i = 0; i < initialCount; i++) {
      initialBoxes.push(await images.nth(i).boundingBox());
    }

    // Rotate to landscape
    await page.setViewportSize({ width: 852, height: 393 });
    await page.waitForTimeout(500); // Allow layout to adjust

    // Verify content still displays properly
    const imagesAfterRotation = page.locator('[data-testid="image-container"] img');
    const afterCount = await imagesAfterRotation.count();
    expect(afterCount).toEqual(initialCount);

    // All images should still be visible
    for (let i = 0; i < afterCount; i++) {
      await expect(imagesAfterRotation.nth(i)).toBeVisible();
    }

    // Layout should have adapted to new orientation
    const newBoxes = [];
    for (let i = 0; i < afterCount; i++) {
      newBoxes.push(await imagesAfterRotation.nth(i).boundingBox());
    }

    // At least some elements should have different positions (layout changed)
    let positionsChanged = false;
    for (let i = 0; i < initialBoxes.length; i++) {
      if (Math.abs(initialBoxes[i]!.x - newBoxes[i]!.x) > 10 ||
          Math.abs(initialBoxes[i]!.y - newBoxes[i]!.y) > 10) {
        positionsChanged = true;
        break;
      }
    }
    expect(positionsChanged).toBeTruthy();
  });

  test('CSS Grid and Flexbox modern layouts', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Find the main image container
    const imageGrid = page.locator('[data-testid="images-grid"]').or(
      page.locator('.images-container').or(
        page.locator('main > div').first()
      )
    );

    if (await imageGrid.isVisible()) {
      const display = await imageGrid.evaluate(el =>
        window.getComputedStyle(el).display
      );

      // Should use modern CSS layout methods
      expect(['grid', 'flex']).toContain(display);

      if (display === 'grid') {
        // Test CSS Grid properties
        const gridTemplateColumns = await imageGrid.evaluate(el =>
          window.getComputedStyle(el).gridTemplateColumns
        );
        expect(gridTemplateColumns).not.toBe('none');

        const gap = await imageGrid.evaluate(el =>
          window.getComputedStyle(el).gap || window.getComputedStyle(el).gridGap
        );
        expect(gap).not.toBe('normal');
      }

      if (display === 'flex') {
        // Test Flexbox properties
        const flexWrap = await imageGrid.evaluate(el =>
          window.getComputedStyle(el).flexWrap
        );
        // Should allow wrapping for responsive behavior
        expect(['wrap', 'wrap-reverse']).toContain(flexWrap);
      }
    }

    // Test that individual image containers use modern CSS
    const imageContainers = page.locator('[data-testid="image-container"]');
    if (await imageContainers.count() > 0) {
      const firstContainer = imageContainers.first();
      const containerDisplay = await firstContainer.evaluate(el =>
        window.getComputedStyle(el).display
      );

      // Container should use modern display methods
      expect(['block', 'flex', 'grid', 'inline-block']).toContain(containerDisplay);
    }
  });

  test('Touch interactions work properly on mobile', async ({ page, browserName }) => {
    // Skip on browsers that don't support touch well in testing
    if (browserName === 'webkit') {
      test.skip();
    }

    // Set mobile viewport with touch
    await page.setViewportSize({ width: 393, height: 852 });

    await page.addInitScript(() => {
      // Mock touch support
      Object.defineProperty(navigator, 'maxTouchPoints', {
        writable: false,
        value: 5
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Test tapping on images
    const images = page.locator('[data-testid="image-container"] img');
    const firstImage = images.first();

    if (await firstImage.isVisible()) {
      // Touch tap should not cause issues
      await firstImage.tap();
      await page.waitForTimeout(300);

      // Image should still be visible and page functional
      await expect(firstImage).toBeVisible();
    }

    // Test any interactive buttons with touch
    const buttons = page.locator('button, [role="button"]');
    const buttonCount = await buttons.count();

    for (let i = 0; i < Math.min(3, buttonCount); i++) {
      const button = buttons.nth(i);
      if (await button.isVisible()) {
        // Tap button
        await button.tap();
        await page.waitForTimeout(200);

        // Page should remain functional
        const main = page.locator('main');
        await expect(main).toBeVisible();
      }
    }

    // Test scroll behavior on touch
    await page.touchscreen.tap(200, 400);

    // Swipe down
    await page.touchscreen.tap(200, 400);
    await page.mouse.move(200, 400);
    await page.mouse.down();
    await page.mouse.move(200, 500);
    await page.mouse.up();

    // Page should still be functional after touch interactions
    await expect(images.first()).toBeVisible();
  });

  test('High DPI and zoom level support', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Test different device pixel ratios
    const dprValues = [1, 1.5, 2, 3];

    for (const dpr of dprValues) {
      await page.emulateMedia({ reducedMotion: 'reduce' });
      await page.addInitScript(dpr => {
        Object.defineProperty(window, 'devicePixelRatio', {
          writable: false,
          value: dpr
        });
      }, dpr);

      await page.reload();
      await page.waitForLoadState('networkidle');

      // Images should still be visible and properly rendered
      const images = page.locator('[data-testid="image-container"] img');
      const imageCount = await images.count();

      for (let i = 0; i < imageCount; i++) {
        await expect(images.nth(i)).toBeVisible();
      }
    }

    // Test browser zoom levels
    const zoomLevels = [0.75, 1.0, 1.25, 1.5, 2.0];

    for (const zoom of zoomLevels) {
      await page.evaluate(zoomLevel => {
        (document.body.style as any).zoom = zoomLevel;
      }, zoom);

      await page.waitForTimeout(300);

      // Content should remain accessible and not overflow
      const hasHorizontalScroll = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });

      expect(hasHorizontalScroll).toBeFalsy();

      // Images should still be visible
      const images = page.locator('[data-testid="image-container"] img');
      if (await images.count() > 0) {
        await expect(images.first()).toBeVisible();
      }
    }

    // Reset zoom
    await page.evaluate(() => {
      (document.body.style as any).zoom = 1.0;
    });
  });

  test('Print media query support', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Emulate print media
    await page.emulateMedia({ media: 'print' });

    // Content should still be visible and properly formatted for print
    const main = page.locator('main');
    await expect(main).toBeVisible();

    const images = page.locator('[data-testid="image-container"] img');
    const imageCount = await images.count();

    // Images should be visible in print mode
    for (let i = 0; i < imageCount; i++) {
      await expect(images.nth(i)).toBeVisible();
    }

    // Check that print styles don't break the layout
    const hasHorizontalScroll = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });

    expect(hasHorizontalScroll).toBeFalsy();
  });
});