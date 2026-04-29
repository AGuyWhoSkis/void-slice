/// <reference types="vite/client" />

declare module "*.decl?raw" {
  const content: string;
  export default content;
}
