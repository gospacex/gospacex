package generator

// Generator 脚手架生成器
type Generator struct {
	engine      *TemplateEngine
	templateDir string
}

// NewGenerator creates new generator
func NewGenerator(templateDir, outputDir string) *Generator {
	return &Generator{
		engine:      NewTemplateEngine(outputDir),
		templateDir: templateDir,
	}
}

// Generate generates a complete project
func (g *Generator) Generate(projectName, entityName, tableName string, fields []FieldConfig, datasources DataSourceConfig) error {
	// Set config
	g.engine.config = &GeneratorConfig{
		ProjectName:     projectName,
		EntityName:      entityName,
		EntityNameLower: ToLowerCamelCase(entityName),
		TableName:       tableName,
		Fields:          fields,
		DataSources:     datasources,
	}

	// Load templates
	if err := g.engine.LoadTemplates(g.templateDir); err != nil {
		return err
	}

	// Generate files
	if err := g.engine.Generate(); err != nil {
		return err
	}

	return nil
}

// GenerateFromConfig generates project from config file
func (g *Generator) GenerateFromConfig(configPath string) error {
	// Load config
	if err := g.engine.LoadConfig(configPath); err != nil {
		return err
	}

	// Load templates
	if err := g.engine.LoadTemplates(g.templateDir); err != nil {
		return err
	}

	// Generate files
	if err := g.engine.Generate(); err != nil {
		return err
	}

	return nil
}
