# PM runtime protocol

This is an isolated PM workspace. Poll `inbox/` for one request-ID-named JSON
file at a time. Record that exact inbox path, read only the request's
`context_directory`, and write the complete decision document atomically to
its unique `output_path`.

Preserve every prior reached-lens record from `previous-decision.md` exactly,
including its classification list, and add only the active lens with the
request's exact request ID and review digest. After writing, wait until that
exact request-specific inbox file is removed; a later request uses a different
filename and cannot satisfy this acknowledgement. Then poll for the next file.
Continue until the process is terminated by the controller.
