package sqlid

// config holds the enabled state of each normalization step. Every step is
// enabled by default; options disable individual steps.
type config struct {
	lowercaseEnabled      bool
	uncommentEnabled      bool
	stripSemicolonEnabled bool
	compressEnabled       bool
	newlineEnabled        bool
	rewriteWithEnabled    bool
	stripConstantsEnabled bool
}

// defaults returns the configuration with every normalization step enabled.
func defaults() config {
	return config{
		lowercaseEnabled:      true,
		uncommentEnabled:      true,
		stripSemicolonEnabled: true,
		compressEnabled:       true,
		newlineEnabled:        true,
		rewriteWithEnabled:    true,
		stripConstantsEnabled: true,
	}
}

// step is one normalization stage together with whether it is enabled.
type step struct {
	fn        transform
	isEnabled bool
}

// steps returns the ordered pipeline for the configuration.
func (c config) steps() []step {
	return []step{
		{isEnabled: c.lowercaseEnabled, fn: lower},
		{isEnabled: c.uncommentEnabled, fn: uncomment},
		{isEnabled: c.stripSemicolonEnabled, fn: stripSemicolon},
		{isEnabled: c.compressEnabled, fn: collapse},
		{isEnabled: c.newlineEnabled, fn: appendNewline},
		{isEnabled: c.rewriteWithEnabled, fn: renameWithAliases},
		{isEnabled: c.stripConstantsEnabled, fn: stripConstants},
	}
}

// run applies the enabled steps to the statement in order.
func (c config) run(s Statement) Statement {
	for _, step := range c.steps() {
		if step.isEnabled {
			s = step.fn(s)
		}
	}
	return s
}

// Option configures normalization. The set of options is closed: only the
// option types defined in this package satisfy the interface.
type Option interface {
	apply(config) config
}

// Lowercase toggles case folding (default true).
type Lowercase bool

// Uncomment toggles removal of non-hint C-style comments (default true).
type Uncomment bool

// StripSemicolon toggles removal of a trailing semicolon (default true).
type StripSemicolon bool

// Compress toggles whitespace compression (default true).
type Compress bool

// Newline toggles appending a trailing newline (default true).
type Newline bool

// RewriteWith toggles rewriting WITH-clause aliases to positional tokens
// (default true).
type RewriteWith bool

// StripConstants toggles replacing string and numeric literals with ?
// (default true).
type StripConstants bool

func (o Lowercase) apply(c config) config      { c.lowercaseEnabled = bool(o); return c }
func (o Uncomment) apply(c config) config      { c.uncommentEnabled = bool(o); return c }
func (o StripSemicolon) apply(c config) config { c.stripSemicolonEnabled = bool(o); return c }
func (o Compress) apply(c config) config       { c.compressEnabled = bool(o); return c }
func (o Newline) apply(c config) config        { c.newlineEnabled = bool(o); return c }
func (o RewriteWith) apply(c config) config    { c.rewriteWithEnabled = bool(o); return c }
func (o StripConstants) apply(c config) config { c.stripConstantsEnabled = bool(o); return c }

// Compile-time verification that every option type satisfies Option.
var (
	_ Option = Lowercase(false)
	_ Option = Uncomment(false)
	_ Option = StripSemicolon(false)
	_ Option = Compress(false)
	_ Option = Newline(false)
	_ Option = RewriteWith(false)
	_ Option = StripConstants(false)
)

// Normalize applies the enabled normalization steps to the statement and
// returns the normalized form.
func Normalize(s Statement, options ...Option) Statement {
	cfg := defaults()
	for _, option := range options {
		cfg = option.apply(cfg)
	}
	return cfg.run(s)
}
