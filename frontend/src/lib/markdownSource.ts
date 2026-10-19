// Textareas expose LF newlines even when the canonical source used CRLF/CR.
// Map selection offsets and apply only the changed span, preserving untouched
// source bytes instead of normalizing an entire pasted document on its next edit.
export function displaySource(source: string) {
  return source.replace(/\r\n?/g, "\n");
}

export function sourceOffset(source: string, offset: number) {
  let index = 0;
  for (let displayed = 0; displayed < offset && index < source.length; displayed++) {
    index += source[index] === "\r" && source[index + 1] === "\n" ? 2 : 1;
  }
  return index;
}

export function reconcileInput(source: string, input: string) {
  const displayed = displaySource(source);
  let prefix = 0;
  while (prefix < displayed.length && prefix < input.length && displayed[prefix] === input[prefix])
    prefix++;
  let suffix = 0;
  while (
    suffix < displayed.length - prefix &&
    suffix < input.length - prefix &&
    displayed[displayed.length - suffix - 1] === input[input.length - suffix - 1]
  )
    suffix++;
  return (
    source.slice(0, sourceOffset(source, prefix)) +
    input.slice(prefix, input.length - suffix) +
    source.slice(sourceOffset(source, displayed.length - suffix))
  );
}
