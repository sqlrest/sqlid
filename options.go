package sqlid

// config holds the enabled state of each normalization step. Every step is
// enabled by default; options disable individual steps.
type config struct {
	lowercase      bool
	uncomment      bool
	stripSemicolon bool
	compress       bool
	newline        bool
	rewriteWith    bool
	stripConstants bool
}

// defaults returns the configuration with every normalization step enabled.
func defaults() config {
	return config{
		lowercase:      true,
		uncomment:      true,
		stripSemicolon: true,
		compress:       true,
		newline:        true,
		rewriteWith:    true,
		stripConstants: true,
	}
}

// step is one normalization stage together with whether it is enabled.
type step struct {
	fn      transform
	enabled bool
}

// steps returns the ordered pipeline for the configuration.
func (c config) steps() []step {
	return []step{
		{enabled: c.lowercase, fn: lower},
		{enabled: c.uncomment, fn: uncomment},
		{enabled: c.stripSemicolon, fn: stripSemicolon},
		{enabled: c.compress, fn: collapse},
		{enabled: c.newline, fn: appendNewline},
		{enabled: c.rewriteWith, fn: renameWithAliases},
		{enabled: c.stripConstants, fn: stripConstants},
	}
}

// run applies the enabled steps to the statement in order.
func (c config) run(s Statement) Statement {
	for _, step := range c.steps() {
		if step.enabled {
			s = step.fn(s)
		}
	}
	return s
}

// Option configures normalization. The set of options is closed: only the
// option types defined in this package satisfy the interface.
type Option interface {
	apply(*config)
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

func (o Lowercase) apply(c *config)      { c.lowercase = bool(o) }
func (o Uncomment) apply(c *config)      { c.uncomment = bool(o) }
func (o StripSemicolon) apply(c *config) { c.stripSemicolon = bool(o) }
func (o Compress) apply(c *config)       { c.compress = bool(o) }
func (o Newline) apply(c *config)        { c.newline = bool(o) }
func (o RewriteWith) apply(c *config)    { c.rewriteWith = bool(o) }
func (o StripConstants) apply(c *config) { c.stripConstants = bool(o) }

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
		option.apply(&cfg)
	}
	return cfg.run(s)
}
