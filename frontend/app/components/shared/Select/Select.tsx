import React from 'react';
import { Select } from 'antd';

type ValueObject = {
  value: string | number;
  label: React.ReactNode;
};

interface Props<Value extends ValueObject> {
  options: Value[];
  showSearch?: boolean;
  defaultValue?: string | number | (string | number)[];
  onChange: (value: any, option: any) => void;
  name?: string;
  placeholder?: string;
  className?: string;
  mode?: 'multiple' | 'tags';
  [x: string]: any;
}

export default function CustomSelect<Value extends ValueObject>({
  placeholder = 'Select',
  name = '',
  onChange,
  options,
  showSearch = false,
  defaultValue = '',
  className = '',
  mode,
  styles,
  ...rest
}: Props<Value>) {
  // Handle onChange to maintain compatibility with the original component
  const handleChange = (value: any, option: any) => {
    onChange({ name, value: option });
  };

  // Filter out non-standard HTML attributes that cause warnings
  if ('right' in rest) {
    rest.right = rest.right.toString();
  }
  if ('plain' in rest) {
    delete rest.plain; // Remove 'plain' as it's not a valid HTML attribute
  }
  
  return (
    <Select
      className={className}
      options={options}
      showSearch={showSearch}
      defaultValue={defaultValue}
      onChange={handleChange}
      placeholder={placeholder}
      mode={mode}
      style={styles}
      popupMatchSelectWidth
      {...rest}
    />
  );
}
