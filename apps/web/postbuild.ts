import fs from 'fs';
import path from 'path';

// copy build folder to ../server/static
const staticPath = '../server/static';

// Remove existing static if it exists
if (fs.existsSync(staticPath)) {
	fs.rmSync(staticPath, { recursive: true, force: true });
}

// Copy build folder to static
fs.cpSync('build', staticPath, { recursive: true });

// Fix CSS links in HTML files
function addCssLinksToHtml() {
	const assetsPath = path.join(staticPath, '_app/immutable/assets');
	if (!fs.existsSync(assetsPath)) {
		console.log('⚠️ No assets directory found');
		return;
	}

	// Find CSS files
	const cssFiles = fs
		.readdirSync(assetsPath)
		.filter((file) => file.endsWith('.css'))
		.filter((file) => file.includes('0.')); // Main CSS file

	if (cssFiles.length === 0) {
		console.log('⚠️ No CSS files found');
		return;
	}

	// Update HTML files
	const htmlFiles = ['index.html', 'auth.html'];

	for (const htmlFile of htmlFiles) {
		const htmlPath = path.join(staticPath, htmlFile);
		if (!fs.existsSync(htmlPath)) continue;

		let content = fs.readFileSync(htmlPath, 'utf-8');

		// Check if CSS links are already present
		const hasCssLinks = cssFiles.some((cssFile) => content.includes(cssFile));

		if (!hasCssLinks) {
			// Add CSS links before closing head tag
			const cssLinks = cssFiles
				.map((cssFile) => `\t\t<link rel="stylesheet" href="./_app/immutable/assets/${cssFile}">`)
				.join('\n');

			content = content.replace('</head>', `${cssLinks}\n\t</head>`);

			fs.writeFileSync(htmlPath, content);
			console.log(`✅ Added CSS links to ${htmlFile}`);
		}
	}
}

addCssLinksToHtml();
console.log('✅ Build files copied to static');
