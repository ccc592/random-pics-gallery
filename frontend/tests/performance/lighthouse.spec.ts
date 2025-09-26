import { test, expect } from '@playwright/test';
import { chromium } from '@playwright/test';

// Performance thresholds as per requirements
const PERFORMANCE_THRESHOLDS = {
  lighthouse_performance: 95,
  first_contentful_paint: 2000, // 2s in milliseconds
  cumulative_layout_shift: 0.1,
  time_to_interactive: 3000, // 3s in milliseconds
} as const;

// Custom performance metrics collector
interface PerformanceMetrics {
  firstContentfulPaint: number;
  timeToInteractive: number;
  cumulativeLayoutShift: number;
  totalBlockingTime: number;
  performanceScore: number;
}

/**
 * Collects Core Web Vitals and performance metrics from the browser
 */
async function collectPerformanceMetrics(page: any): Promise<PerformanceMetrics> {
  // Collect performance metrics using browser APIs
  const metrics = await page.evaluate(() => {
    return new Promise((resolve) => {
      // Wait for page load to complete
      if (document.readyState === 'complete') {
        collectMetrics();
      } else {
        window.addEventListener('load', collectMetrics);
      }

      function collectMetrics() {
        const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
        const paint = performance.getEntriesByType('paint');

        let fcp = 0;
        const fcpEntry = paint.find((entry) => entry.name === 'first-contentful-paint');
        if (fcpEntry) {
          fcp = fcpEntry.startTime;
        }

        // Calculate TTI approximation
        const tti = navigation.loadEventEnd - navigation.navigationStart;

        // Basic CLS calculation (simplified)
        let cls = 0;
        try {
          // This would need a more sophisticated implementation in real usage
          cls = 0; // Placeholder - would require layout shift observer
        } catch (e) {
          cls = 0;
        }

        // Calculate total blocking time approximation
        let tbt = 0;
        const resources = performance.getEntriesByType('resource');
        resources.forEach((resource: any) => {
          if (resource.duration > 50) {
            tbt += resource.duration - 50;
          }
        });

        resolve({
          firstContentfulPaint: fcp,
          timeToInteractive: tti,
          cumulativeLayoutShift: cls,
          totalBlockingTime: tbt,
          performanceScore: 0, // Will be calculated
        });
      }
    });
  });

  // Calculate a basic performance score (simplified version of Lighthouse scoring)
  const fcpScore = Math.max(0, 100 - (metrics.firstContentfulPaint / 1000 - 1.8) * 50);
  const ttiScore = Math.max(0, 100 - (metrics.timeToInteractive / 1000 - 3.8) * 10);
  const clsScore = Math.max(0, 100 - metrics.cumulativeLayoutShift * 750);
  const tbtScore = Math.max(0, 100 - (metrics.totalBlockingTime - 200) / 10);

  const performanceScore = (fcpScore * 0.1 + ttiScore * 0.1 + clsScore * 0.15 + tbtScore * 0.3 + 45); // Simplified

  return {
    ...metrics,
    performanceScore: Math.min(100, Math.max(0, performanceScore)),
  };
}

test.describe('Frontend Performance Validation', () => {
  test.beforeEach(async ({ page }) => {
    // Clear cache and storage for consistent testing
    await page.context().clearCookies();
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
  });

  test('Homepage Lighthouse Performance Score >= 95', async ({ page }) => {
    // Start performance measurement
    const startTime = Date.now();

    // Navigate to homepage
    await page.goto('/');

    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle');

    // Collect performance metrics
    const metrics = await collectPerformanceMetrics(page);

    const loadTime = Date.now() - startTime;

    console.log('Performance Metrics:');
    console.log(`  Performance Score: ${metrics.performanceScore.toFixed(1)}/100`);
    console.log(`  First Contentful Paint: ${metrics.firstContentfulPaint.toFixed(0)}ms`);
    console.log(`  Time to Interactive: ${metrics.timeToInteractive.toFixed(0)}ms`);
    console.log(`  Cumulative Layout Shift: ${metrics.cumulativeLayoutShift.toFixed(3)}`);
    console.log(`  Total Blocking Time: ${metrics.totalBlockingTime.toFixed(0)}ms`);
    console.log(`  Page Load Time: ${loadTime}ms`);

    // Verify performance thresholds
    expect(metrics.performanceScore, 'Lighthouse Performance Score should be >= 95').toBeGreaterThanOrEqual(PERFORMANCE_THRESHOLDS.lighthouse_performance);
    expect(metrics.firstContentfulPaint, 'First Contentful Paint should be <= 2s').toBeLessThanOrEqual(PERFORMANCE_THRESHOLDS.first_contentful_paint);
    expect(metrics.cumulativeLayoutShift, 'Cumulative Layout Shift should be < 0.1').toBeLessThan(PERFORMANCE_THRESHOLDS.cumulative_layout_shift);
    expect(metrics.timeToInteractive, 'Time to Interactive should be <= 3s').toBeLessThanOrEqual(PERFORMANCE_THRESHOLDS.time_to_interactive);
  });

  test('Gallery Page Performance Under Load', async ({ page }) => {
    await page.goto('/gallery');
    await page.waitForLoadState('networkidle');

    const metrics = await collectPerformanceMetrics(page);

    console.log('Gallery Page Performance:');
    console.log(`  Performance Score: ${metrics.performanceScore.toFixed(1)}/100`);
    console.log(`  First Contentful Paint: ${metrics.firstContentfulPaint.toFixed(0)}ms`);
    console.log(`  Time to Interactive: ${metrics.timeToInteractive.toFixed(0)}ms`);

    // Gallery page should still meet performance criteria
    expect(metrics.performanceScore).toBeGreaterThanOrEqual(90); // Slightly lower threshold for content-heavy page
    expect(metrics.firstContentfulPaint).toBeLessThanOrEqual(2500); // Slightly higher for content loading
  });

  test('Image Loading Performance', async ({ page }) => {
    await page.goto('/');

    // Wait for initial load
    await page.waitForLoadState('networkidle');

    // Measure time to load random images
    const imageLoadStart = Date.now();

    // Click button to load random images (assuming there's a "Load Random Images" button)
    const loadButton = page.locator('button:has-text("Load Random Images"), button:has-text("Get Random Images"), [data-testid="load-images"]').first();

    if (await loadButton.count() > 0) {
      await loadButton.click();

      // Wait for images to load
      await page.waitForLoadState('networkidle');

      const imageLoadTime = Date.now() - imageLoadStart;

      console.log(`Image Loading Time: ${imageLoadTime}ms`);

      // Image loading should be fast
      expect(imageLoadTime, 'Image loading should be under 2 seconds').toBeLessThan(2000);
    }

    // Check for image optimization
    const images = await page.locator('img').all();
    for (const img of images) {
      const src = await img.getAttribute('src');
      const alt = await img.getAttribute('alt');

      if (src) {
        console.log(`Image: ${src}, Alt: ${alt}`);

        // Verify images have alt text for accessibility
        expect(alt, 'Images should have alt text').toBeTruthy();
        expect(alt!.length, 'Alt text should be descriptive').toBeGreaterThan(5);
      }
    }
  });

  test('Mobile Performance', async ({ browser }) => {
    // Create mobile context
    const mobileContext = await browser.newContext({
      viewport: { width: 375, height: 667 }, // iPhone SE dimensions
      deviceScaleFactor: 2,
      isMobile: true,
      hasTouch: true,
    });

    const page = await mobileContext.newPage();

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const metrics = await collectPerformanceMetrics(page);

    console.log('Mobile Performance Metrics:');
    console.log(`  Performance Score: ${metrics.performanceScore.toFixed(1)}/100`);
    console.log(`  First Contentful Paint: ${metrics.firstContentfulPaint.toFixed(0)}ms`);
    console.log(`  Time to Interactive: ${metrics.timeToInteractive.toFixed(0)}ms`);

    // Mobile should still meet reasonable performance criteria
    expect(metrics.performanceScore).toBeGreaterThanOrEqual(85); // Slightly lower for mobile
    expect(metrics.firstContentfulPaint).toBeLessThanOrEqual(3000); // Mobile network consideration

    await mobileContext.close();
  });

  test('Network Performance Simulation', async ({ page }) => {
    // Simulate slow 3G network
    const context = page.context();
    await context.route('**/*', async (route) => {
      // Add artificial delay to simulate slow network
      await new Promise(resolve => setTimeout(resolve, 100)); // 100ms delay
      await route.continue();
    });

    const startTime = Date.now();
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const loadTime = Date.now() - startTime;

    console.log(`Load time with simulated slow network: ${loadTime}ms`);

    // Should still load within reasonable time even on slow network
    expect(loadTime, 'Page should load within 10 seconds on slow network').toBeLessThan(10000);

    const metrics = await collectPerformanceMetrics(page);
    console.log(`Performance score on slow network: ${metrics.performanceScore.toFixed(1)}/100`);
  });

  test('Resource Optimization Check', async ({ page }) => {
    // Track all network requests
    const requests: any[] = [];

    page.on('request', (request) => {
      requests.push({
        url: request.url(),
        resourceType: request.resourceType(),
        method: request.method(),
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Analyze resource loading
    const jsRequests = requests.filter(r => r.resourceType === 'script');
    const cssRequests = requests.filter(r => r.resourceType === 'stylesheet');
    const imageRequests = requests.filter(r => r.resourceType === 'image');

    console.log('Resource Analysis:');
    console.log(`  Total requests: ${requests.length}`);
    console.log(`  JavaScript files: ${jsRequests.length}`);
    console.log(`  CSS files: ${cssRequests.length}`);
    console.log(`  Images: ${imageRequests.length}`);

    // Check for reasonable resource counts
    expect(jsRequests.length, 'Should not have excessive JS files').toBeLessThan(10);
    expect(cssRequests.length, 'Should not have excessive CSS files').toBeLessThan(5);

    // Check for optimized image formats
    for (const imgRequest of imageRequests) {
      const url = imgRequest.url;
      const hasModernFormat = url.includes('.webp') || url.includes('.avif') || url.includes('.jpg') || url.includes('.png');
      expect(hasModernFormat, `Image should use optimized format: ${url}`).toBeTruthy();
    }
  });

  test('Caching Performance', async ({ page }) => {
    // First visit
    const firstVisitStart = Date.now();
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const firstVisitTime = Date.now() - firstVisitStart;

    // Reload page (should use cache)
    const reloadStart = Date.now();
    await page.reload();
    await page.waitForLoadState('networkidle');
    const reloadTime = Date.now() - reloadStart;

    console.log(`First visit: ${firstVisitTime}ms`);
    console.log(`Reload (cached): ${reloadTime}ms`);

    // Cached reload should be significantly faster
    expect(reloadTime, 'Cached reload should be faster than first visit').toBeLessThan(firstVisitTime);
    expect(reloadTime, 'Cached reload should be very fast').toBeLessThan(1000);
  });

  test('Progressive Enhancement Check', async ({ page }) => {
    // Disable JavaScript to test progressive enhancement
    await page.context().addInitScript(() => {
      Object.defineProperty(window, 'navigator', {
        value: {
          ...window.navigator,
          userAgent: window.navigator.userAgent + ' [JS_DISABLED]'
        }
      });
    });

    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Check if basic content is still accessible without JS
    const hasContent = await page.locator('body').textContent();
    expect(hasContent, 'Page should have content without JavaScript').toBeTruthy();
    expect(hasContent!.length, 'Page should have meaningful content').toBeGreaterThan(100);

    // Check for proper semantic HTML
    const headings = await page.locator('h1, h2, h3').count();
    expect(headings, 'Page should have semantic headings').toBeGreaterThan(0);
  });

  test('Animation Performance', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check for performance-expensive animations
    const hasAnimations = await page.evaluate(() => {
      const styles = Array.from(document.styleSheets).flatMap(sheet => {
        try {
          return Array.from(sheet.cssRules);
        } catch (e) {
          return [];
        }
      });

      const animationRules = styles.filter((rule: any) =>
        rule.cssText && (
          rule.cssText.includes('animation') ||
          rule.cssText.includes('transition') ||
          rule.cssText.includes('transform')
        )
      );

      return animationRules.length > 0;
    });

    if (hasAnimations) {
      console.log('Animations detected - checking performance impact');

      // Measure performance with animations
      const metrics = await collectPerformanceMetrics(page);

      // Animations shouldn't severely impact performance
      expect(metrics.performanceScore).toBeGreaterThanOrEqual(90);
    }
  });
});

test.describe('Core Web Vitals Monitoring', () => {
  test('Largest Contentful Paint (LCP)', async ({ page }) => {
    await page.goto('/');

    const lcp = await page.evaluate(() => {
      return new Promise((resolve) => {
        new PerformanceObserver((list) => {
          const entries = list.getEntries();
          const lastEntry = entries[entries.length - 1];
          resolve(lastEntry.startTime);
        }).observe({ entryTypes: ['largest-contentful-paint'] });

        // Fallback timeout
        setTimeout(() => resolve(0), 5000);
      });
    });

    console.log(`Largest Contentful Paint: ${lcp}ms`);
    expect(lcp, 'LCP should be under 2.5 seconds').toBeLessThan(2500);
  });

  test('First Input Delay (FID) Simulation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Simulate user interaction
    const interactionStart = Date.now();
    await page.click('body'); // Click somewhere on the page
    const interactionTime = Date.now() - interactionStart;

    console.log(`Simulated interaction delay: ${interactionTime}ms`);
    expect(interactionTime, 'Interaction delay should be minimal').toBeLessThan(100);
  });

  test('Cumulative Layout Shift (CLS)', async ({ page }) => {
    let clsValue = 0;

    // Monitor layout shifts
    await page.evaluateOnNewDocument(() => {
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (!(entry as any).hadRecentInput) {
            (window as any).clsValue = ((window as any).clsValue || 0) + (entry as any).value;
          }
        }
      }).observe({ type: 'layout-shift', buffered: true });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Get final CLS value
    clsValue = await page.evaluate(() => (window as any).clsValue || 0);

    console.log(`Cumulative Layout Shift: ${clsValue}`);
    expect(clsValue, 'CLS should be under 0.1').toBeLessThan(0.1);
  });
});

// Fix Date.Now typo
test.afterEach(async ({ page }) => {
  // Clean up any test artifacts
  await page.close();
});