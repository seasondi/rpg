package engine

import (
	"fmt"
	"github.com/beevik/etree"
)

func initEntityDefs() error {
	if err := initDataTypes(); err != nil {
		return err
	}
	defMgr = new(entityDefs)
	if err := defMgr.Init(); err != nil {
		return err
	}

	log.Infof("entity defs inited.")
	return nil
}

type entityDefs struct {
	defMap map[string]*entityDef
	alias  map[string]propType
}

func (m *entityDefs) Init() error {
	if err := m.LoadAlias(); err != nil {
		return err
	}
	if err := m.LoadEntityDef(); err != nil {
		return err
	}
	return nil
}

func (m *entityDefs) LoadAlias() error {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(cfg.WorkPath + "/defs/alias.xml"); err != nil {
		log.Errorf("read alias.xml failed, msg: %s", err.Error())
		return err
	}
	m.alias = make(map[string]propType)
	root := doc.SelectElement(defFieldRoot)
	for _, tp := range root.ChildElements() {
		m.alias[tp.Tag] = readPropType(tp, tp.Tag, readTypeAlias)
	}
	return nil
}

func (m *entityDefs) LoadEntityDef() error {
	m.defMap = make(map[string]*entityDef)
	entityXmlFile := cfg.WorkPath + "/defs/entities.xml"
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(entityXmlFile); err != nil {
		log.Errorf("read file[%s] error: %s", entityXmlFile, err.Error())
		return err
	}
	root := doc.SelectElement("root")
	if root == nil {
		log.Errorf("[%s] must start with \"root\"", entityXmlFile)
		return fmt.Errorf("[%s] must start with \"root\"", entityXmlFile)
	}
	for _, ent := range root.SelectElements("entity") {
		entDef := new(entityDef)
		entDef.Load(ent.Text())
		m.defMap[entDef.entityName] = entDef
	}
	if entryEntityName == "" {
		log.Errorf("not found entry method[%s] in all def files", StubEntryMethod)
		return fmt.Errorf("not found entry method[%s] in all def files", StubEntryMethod)
	}
	return nil
}

func (m *entityDefs) GetEntityDef(name string) *entityDef {
	return m.defMap[name]
}

func (m *entityDefs) GetAlias(name string) *propType {
	if pt, ok := m.alias[name]; ok {
		return &pt
	}
	return nil
}

func (m *entityDefs) GetInterfaces(name string) []string {
	def := m.defMap[name]
	if def == nil {
		return []string{}
	} else {
		return def.interfaces
	}
}
