export type CompilerLanguage = "python" | "java" | "c" | "cpp";

export interface LanguageConfig {
  key: CompilerLanguage;
  label: string;
  monacoLanguage: string;
  filename: string;
  starterCode: string;
}

export const REPOSITORY_URL = "https://github.com/Anga205/CodeExecutionAPI";

const DEFAULT_ANGADRIVE_BASE_URL = "https://i.anga.codes";
const DEFAULT_EXECUTE_ENDPOINT = "https://codeapi.anga.codes/v2/execute";

export const EXECUTE_ENDPOINT =
  import.meta.env.PUBLIC_EXECUTE_ENDPOINT || DEFAULT_EXECUTE_ENDPOINT;

export const NANOSECONDS_PER_SECOND = 1_000_000_000;

export const LIMITS = {
  defaultTimeoutSeconds: 5,
  minTimeoutSeconds: 1,
  maxTimeoutSeconds: 60,
  defaultMemoryMb: 32,
  minMemoryMb: 8,
  maxMemoryMb: 128
} as const;

export const LANGUAGES: LanguageConfig[] = [
  {
    key: "python",
    label: "Python",
    monacoLanguage: "python",
    filename: "main.py",
    starterCode: 'print("Hello, World!")'
  },
  {
    key: "c",
    label: "C",
    monacoLanguage: "c",
    filename: "main.c",
    starterCode: [
      "#include <stdio.h>",
      "",
      "int main(void) {",
      '    printf("Hello, World!\\n");',
      "    return 0;",
      "}"
    ].join("\n")
  },
  {
    key: "cpp",
    label: "C++",
    monacoLanguage: "cpp",
    filename: "main.cpp",
    starterCode: [
      "#include <iostream>",
      "",
      "int main() {",
      '    std::cout << "Hello, World!" << std::endl;',
      "    return 0;",
      "}"
    ].join("\n")
  },
  {
    key: "java",
    label: "Java",
    monacoLanguage: "java",
    filename: "Main.java",
    starterCode: [
      "public class Main {",
      "    public static void main(String[] args) {",
      '        System.out.println("Hello, World!");',
      "    }",
      "}"
    ].join("\n")
  }
];

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

export const getLanguageConfig = (language: string): LanguageConfig => {
  return LANGUAGES.find((entry) => entry.key === language) ?? LANGUAGES[0];
};

export const clampTimeout = (seconds: number): number => {
  if (!Number.isFinite(seconds)) return LIMITS.defaultTimeoutSeconds;
  return Math.min(
    Math.max(Math.round(seconds), LIMITS.minTimeoutSeconds),
    LIMITS.maxTimeoutSeconds
  );
};

export const clampMemory = (megabytes: number): number => {
  if (!Number.isFinite(megabytes)) return LIMITS.defaultMemoryMb;
  return Math.min(
    Math.max(Math.round(megabytes), LIMITS.minMemoryMb),
    LIMITS.maxMemoryMb
  );
};

const toCompactNumber = (value: number, fractionDigits = 2) => {
  return Number(value.toFixed(fractionDigits)).toString();
};

const parseCpuTimeToNanoseconds = (cpuTime: unknown): number | null => {
  if (typeof cpuTime === "number" && Number.isFinite(cpuTime) && cpuTime >= 0) {
    return cpuTime;
  }

  if (typeof cpuTime !== "string") {
    return null;
  }

  const normalized = cpuTime.trim().toLowerCase();
  if (!normalized) {
    return null;
  }

  const match = normalized.match(/^([\d.]+)\s*(ns|us|µs|ms|s|m)?$/);
  if (!match) {
    return null;
  }

  const value = Number(match[1]);
  const unit = match[2] ?? "ns";
  if (!Number.isFinite(value) || value < 0) {
    return null;
  }

  if (unit === "ns") return value;
  if (unit === "us" || unit === "µs") return value * 1_000;
  if (unit === "ms") return value * 1_000_000;
  if (unit === "s") return value * 1_000_000_000;
  if (unit === "m") return value * 60 * 1_000_000_000;

  return null;
};

export const formatCpuTime = (cpuTime: unknown): string => {
  const nanoseconds = parseCpuTimeToNanoseconds(cpuTime);
  if (nanoseconds == null) {
    return "—";
  }

  if (nanoseconds < 1_000) {
    return `${toCompactNumber(nanoseconds, 0)} ns`;
  }

  if (nanoseconds < 1_000_000) {
    return `${toCompactNumber(nanoseconds / 1_000)} µs`;
  }

  if (nanoseconds < 1_000_000_000) {
    return `${toCompactNumber(nanoseconds / 1_000_000)} ms`;
  }

  if (nanoseconds < 60 * 1_000_000_000) {
    return `${toCompactNumber(nanoseconds / 1_000_000_000)} s`;
  }

  return `${toCompactNumber(nanoseconds / (60 * 1_000_000_000))} min`;
};

export const formatMemory = (memoryUsed: unknown): string => {
  if (typeof memoryUsed === "number" && Number.isFinite(memoryUsed) && memoryUsed > 0) {
    return `${(memoryUsed / 1024).toFixed(2)} MB`;
  }

  if (typeof memoryUsed === "string") {
    const parsed = Number(memoryUsed.replace(/[^\d.]/g, ""));
    if (Number.isFinite(parsed) && parsed > 0) {
      return `${(parsed / 1024).toFixed(2)} MB`;
    }
  }

  return "—";
};

export interface CompilerEditorApi {
  getCode: () => string;
  getStdin: () => string;
  focus: () => void;
}

declare global {
  interface Window {
    __compilerEditorApi?: CompilerEditorApi;
  }
}
