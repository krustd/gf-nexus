import React from 'react';
import Editor from '@monaco-editor/react';
import type { ConfigFormat } from '@/types';

interface ConfigEditorProps {
  value: string;
  format: ConfigFormat;
  onChange?: (value: string | undefined) => void;
  readOnly?: boolean;
  height?: string | number;
  theme?: 'vs-dark' | 'light';
}

// 格式到 Monaco 语言的映射
const formatToLanguage: Record<ConfigFormat, string> = {
  yaml: 'yaml',
  json: 'json',
  toml: 'ini', // Monaco 没有 toml，用 ini 代替
  properties: 'ini',
};

const ConfigEditor: React.FC<ConfigEditorProps> = ({
  value,
  format,
  onChange,
  readOnly = false,
  height = '500px',
  theme = 'vs-dark',
}) => {
  const language = formatToLanguage[format];

  return (
    <Editor
      height={height}
      language={language}
      value={value}
      onChange={onChange}
      theme={theme}
      options={{
        readOnly,
        minimap: { enabled: false },
        fontSize: 14,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        automaticLayout: true,
        tabSize: 2,
        wordWrap: 'on',
      }}
    />
  );
};

export default ConfigEditor;
