import * as React from "react";
import AutocompleteInput from "./AutocompleteInput";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  values: string[];
  onValuesChange: (values: string[]) => void;
  placeholder?: string;
  label?: string;
};

export default function ContextsAutocomplete({ values, onValuesChange, placeholder = "Add context...", label }: Props) {
  const [suggestions, setSuggestions] = React.useState<string[]>([]);

  React.useEffect(() => {
    Backend.GetFilters().then((result: any) => {
      if (result && typeof result === 'object') {
        setSuggestions(result.contexts || []);
      }
    }).catch(err => {
      console.error("Failed to load context filters:", err);
      setSuggestions([]);
    });
  }, []);

  return (
    <AutocompleteInput
      values={values}
      onValuesChange={onValuesChange}
      suggestions={suggestions}
      placeholder={placeholder}
      prefix="@"
      label={label}
    />
  );
}
