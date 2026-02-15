import React from 'react';
import { DiffEditor } from '@monaco-editor/react';
import type { ConfigFormat } from '@/types';

interface DiffViewerProps {
  originalValue: string;
  modifiedValue: string;
  format: ConfigFormat;
  originalTitle?: string;
  modifiedTitle?: string;
  height?: string | number;
  theme?: 'vs-dark' | 'light';
  readOnly?: boolean;
}

// 格式到 Monaco 语言的映射
const formatToLanguage: Record<ConfigFormat, string> = {
  yaml: 'yaml',
  json: 'json',
  toml: 'ini',
  properties: 'ini',
};

const DiffViewer: React.FC<DiffViewerProps> = ({
  originalValue,
  modifiedValue,
  format,
  originalTitle = '已发布版本',
  modifiedTitle = '草稿版本',
  height = '600px',
  theme = 'vs-dark',
  readOnly = true,
}) => {
  const language = formatToLanguage[format];

  return (
    <div>
      <div style={{
        display: 'flex',
        justifyContent: 'space-around',
        padding: '8px 0',
        backgroundColor: theme === 'vs-dark' ? '#1e1e1e' : '#f5f5f5',
        color: theme === 'vs-dark' ? '#fff' : '#000',
      }}>
        <div>{originalTitle}</div>
        <div>{modifiedTitle}</div>
      </div>
      <DiffEditor
        height={height}
        language={language}
        original={originalValue}
        modified={modifiedValue}
        theme={theme}
        options={{
          readOnly,
          renderSideBySide: true,
          minimap: { enabled: false },
          fontSize: 14,
          scrollBeyondLastLine: false,
          automaticLayout: true,
        }}
      />
    </div>
  );
};

export default DiffViewer;
