import * as React from "react";
import AutocompleteInput from "./AutocompleteInput";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  values: string[];
  onValuesChange: (values: string[]) => void;
  placeholder?: string;
  label?: string;
};

export default function TagsAutocomplete({ values, onValuesChange, placeholder = "Add tag...", label }: Props) {
  const [suggestions, setSuggestions] = React.useState<string[]>([]);

  React.useEffect(() => {
    Backend.GetFilters().then((result: any) => {
      if (result && typeof result === 'object') {
        setSuggestions(result.tags || []);
      }
    }).catch(err => {
      console.error("Failed to load tag filters:", err);
      setSuggestions([]);
    });
  }, []);

  return (
    <AutocompleteInput
      values={values}
      onValuesChange={onValuesChange}
      suggestions={suggestions}
      placeholder={placeholder}
      prefix="#"
      {...(label !== undefined && { label })}
    />
  );
}
