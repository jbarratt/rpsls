import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';

export default {
  input: 'src/main.js',
  output: {
    file: 'public/bundle.js',
    formate: 'iife',
    sourcemap: true
  },
  plugins: [
    resolve(),
    commonjs(),
    // terser() for production
  ]
};
