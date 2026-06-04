const { glob } = require('glob');
const readFile = require('./readFile');

module.exports = async (sourcemap_dir_path, js_dir_url) => {
  sourcemap_dir_path = (sourcemap_dir_path + '/').replace(/\/+/g, '/');
  if (js_dir_url[js_dir_url.length - 1] !== '/') {
    // replace will break schema
    js_dir_url += '/';
  }
  const sourcemap_file_paths = await glob(sourcemap_dir_path + '**/*.map');
  return Promise.all(
    sourcemap_file_paths.map((sourcemap_file_path) =>
      readFile(
        sourcemap_file_path,
        js_dir_url + sourcemap_file_path.slice(sourcemap_dir_path.length, -4),
      ),
    ),
  );
};
