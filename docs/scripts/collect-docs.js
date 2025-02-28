import { readFile, writeFile, mkdir, copyFile as fsCopyFile } from 'fs/promises';
import { dirname, join, basename, extname } from 'path';
import { fileURLToPath } from 'url';
import { glob } from 'glob';
import { watch } from 'fs';
import { exec } from 'child_process';
import { promisify } from 'util';
import sharp from 'sharp';

const execAsync = promisify(exec);
const __dirname = dirname(fileURLToPath(import.meta.url));
const rootDir = join(__dirname, '../../');
const docsDir = join(__dirname, '../src/content/docs');
const assetsDir = join(__dirname, '../src/content/docs/assets');

// Patterns to ignore
const ignorePatterns = [
  '**/node_modules/**',
  '**/docs/**',
  '**/dist/**',
  '**/vendor/**',
  '**/build/**',
  '**/tmp/**'
];

function generateTitle(filePath) {
  // Получаем имя директории, в которой находится README
  const dir = dirname(filePath);
  if (dir === '.') return 'Root Documentation';
  
  // Разбиваем путь на части и берем последнюю директорию
  const parts = dir.split('/');
  const lastDir = parts[parts.length - 1];
  
  // Преобразуем kebab-case или snake_case в Title Case
  return lastDir
    .replace(/[-_]/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2') // Разделяем camelCase
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
}

function generateDescription(filePath) {
  // Создаем описание на основе пути к файлу
  const parts = filePath.split('/').filter(part => part !== 'README.md');
  if (parts.length === 0) return 'Main project documentation';
  
  return `Documentation for the ${parts.join(' ')} component`;
}

function addFrontMatter(content, filePath) {
  const title = generateTitle(filePath);
  const description = generateDescription(filePath);
  
  const frontMatter = `---
title: ${title}
description: ${description}
---

`;

  // Check if frontmatter already exists
  if (content.startsWith('---')) {
    // If exists, leave as is
    return content;
  }
  
  return frontMatter + content;
}

async function copyFile(file) {
  const sourcePath = join(rootDir, file);
  const targetDir = join(docsDir, dirname(file));
  const targetPath = join(targetDir, 'index.md');

  try {
    // Create target directory
    await mkdir(targetDir, { recursive: true });

    // Read the file
    const content = await readFile(sourcePath, 'utf8');
    
    // Add frontmatter if needed
    const processedContent = addFrontMatter(content, file);

    // Write the processed content
    await writeFile(targetPath, processedContent);

    console.log(`Copied and processed ${file} to ${targetPath}`);
  } catch (error) {
    console.error(`Error copying file ${file}:`, error);
  }
}

async function processAndCopyImage(sourcePath, targetPath) {
  try {
    const ext = extname(sourcePath).toLowerCase();
    const isImage = ['.png', '.jpg', '.jpeg', '.gif', '.webp'].includes(ext);

    if (!isImage) {
      // If not an image, just copy the file
      await fsCopyFile(sourcePath, targetPath);
      return;
    }

    // Process image
    await sharp(sourcePath)
      .resize(1200, 1200, { // Maximum dimensions
        fit: 'inside', // Preserve aspect ratio
        withoutEnlargement: true // Don't enlarge small images
      })
      .toFile(targetPath);

    console.log(`Processed and copied image ${sourcePath} to ${targetPath}`);
  } catch (error) {
    console.error(`Error processing image ${sourcePath}:`, error);
    // If processing failed, try to just copy the file
    await fsCopyFile(sourcePath, targetPath);
  }
}

async function copyAssets() {
  try {
    // Find all files in assets directory
    const assetFiles = await glob('**/assets/**/*.*', {
      cwd: rootDir,
      ignore: ignorePatterns,
      nocase: true,
    });

    console.log('Found asset files:', assetFiles);

    for (const file of assetFiles) {
      const sourcePath = join(rootDir, file);
      const targetPath = join(assetsDir, basename(file));

      try {
        // Create assets directory if it doesn't exist
        await mkdir(dirname(targetPath), { recursive: true });
        
        // Process and copy the asset file
        await processAndCopyImage(sourcePath, targetPath);
      } catch (error) {
        console.error(`Error copying asset ${file}:`, error);
      }
    }
  } catch (error) {
    console.error('Error collecting assets:', error);
  }
}

async function collectPluginsDocs() {
  try {
    // First get the list of all plugins
    const { stdout: pluginsList } = await execAsync('go run main.go plugins', { cwd: rootDir });
    const plugins = pluginsList
      .split('\n')
      .filter(line => line.trim() && !line.includes('Available Plugins:'))
      .map(line => line.trim());

    console.log('Found plugins:', plugins);

    // Create directory for plugins
    const pluginsDir = join(docsDir, 'plugins');
    await mkdir(pluginsDir, { recursive: true });

    // Collect documentation for each plugin
    for (const plugin of plugins) {
      const { stdout: pluginDoc } = await execAsync(`go run main.go plugins ${plugin}`, { cwd: rootDir });
      
      // Create documentation file for the plugin
      const pluginPath = join(pluginsDir, `${plugin}.md`);
      
      const content = `---
title: ${plugin} Plugin
description: Documentation for the ${plugin} plugin
---

${pluginDoc}`;

      await writeFile(pluginPath, content);
      console.log(`Generated documentation for plugin ${plugin}`);
    }

    // Create index file for plugins
    const indexContent = `---
title: Plugins
description: List of all available plugins and their documentation
---

# Available Plugins

${plugins.map(plugin => `- [${plugin}](${plugin})`).join('\n')}
`;

    await writeFile(join(pluginsDir, 'index.md'), indexContent);
    console.log('Generated plugins index');

  } catch (error) {
    console.error('Error collecting plugins documentation:', error);
  }
}

async function collectConnectorsDocs() {
  try {
    const connectorsPath = join(rootDir, 'connectors');
    const files = await glob('**/README.md', {
      cwd: connectorsPath,
      nocase: true,
    });

    for (const file of files) {
      const sourcePath = join(connectorsPath, file);
      const content = await readFile(sourcePath, 'utf8');
      
      // Convert path: connectors/foo/README.md -> connectors/foo.md
      const targetPath = join(docsDir, 'connectors', 
        file.toLowerCase() === 'readme.md' 
          ? 'index.md'
          : `${dirname(file)}.md`
      );
      
      await mkdir(dirname(targetPath), { recursive: true });
      await writeFile(targetPath, addFrontMatter(content, file));
      
      console.log(`Processed connector documentation: ${file} -> ${targetPath}`);
    }
  } catch (error) {
    console.error('Error collecting connectors documentation:', error);
  }
}

async function collectDocs() {
  try {
    // Find all README.md files in the project
    const files = await glob('**/*.md', {
      cwd: rootDir,
      ignore: [...ignorePatterns, '**/connectors/**', '**/plugins/**'],
      nocase: true, // Case-insensitive search
    });

    console.log('Found files:', files);

    for (const file of files) {
      await copyFile(file);
    }

    // Copy assets
    await copyAssets();

    // Collect plugins and connectors documentation
    await collectPluginsDocs();
    await collectConnectorsDocs();

    console.log('Documentation collection completed!');
    return files;
  } catch (error) {
    console.error('Error collecting documentation:', error);
    process.exit(1);
  }
}

function shouldProcessFile(filepath) {
  // Check if file is README.md and not in ignored directories
  const isReadme = /readme\.md$/i.test(filepath);
  const isIgnored = ignorePatterns.some(pattern => {
    const regexPattern = pattern.replace(/\*\*/g, '.*');
    return new RegExp(regexPattern, 'i').test(filepath);
  });
  
  return isReadme && !isIgnored;
}

async function watchFiles() {
  // Сначала собираем все файлы
  const initialFiles = await collectDocs();
  
  console.log('Watching for file changes...');
  
  // Начинаем отслеживать изменения
  watch(rootDir, { recursive: true }, async (eventType, filename) => {
    if (!filename) return;
    
    const relativePath = filename.replace(/\\/g, '/'); // Нормализуем путь для Windows
    
    if (shouldProcessFile(relativePath)) {
      console.log(`Change detected in ${relativePath}`);
      await copyFile(relativePath);
    }
  });
}

// Проверяем аргументы командной строки
const args = process.argv.slice(2);
if (args.includes('--watch')) {
  watchFiles();
} else {
  collectDocs();
} 
