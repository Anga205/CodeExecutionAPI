export type CompilerLanguage = "python" | "java" | "c" | "cpp";

const DEFAULT_ANGADRIVE_BASE_URL = "https://i.anga.codes";

const languageByExtension: Record<string, CompilerLanguage> = {
  py: "python",
  java: "java",
  c: "c",
  cc: "cpp",
  cpp: "cpp",
  cxx: "cpp"
};

export const getAngaDriveFileUrl = (filename: string) => {
  const baseUrl = import.meta.env.ANGADRIVE_BASE_URL || DEFAULT_ANGADRIVE_BASE_URL;
  return new URL(`i/${encodeURIComponent(filename)}`, `${baseUrl.replace(/\/+$/, "")}/`);
};

export const inferCompilerLanguage = (filename: string): CompilerLanguage => {
  const extension = filename.split(".").pop()?.toLowerCase() ?? "";
  return languageByExtension[extension] ?? "python";
};
