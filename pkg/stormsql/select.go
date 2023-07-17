package stormsql

import (
	"fmt"
	"strconv"

	"github.com/araddon/dateparse"
	"github.com/asdine/storm/v3/q"
	"github.com/pkg/errors"
	"github.com/xwb1989/sqlparser"
)

// A SelectClause contains all the parsed SQL data.
type SelectClause struct {
	SelectedFields  []string
	Count           bool
	Tablename       string
	Matcher         q.Matcher
	Skip            int
	Limit           int
	OrderBy         []string
	OrderByReversed bool
}

// ParseSelect parses the given SELECT statement.
func ParseSelect(sql string) (*SelectClause, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse SQL")
	}

	s, ok := stmt.(*sqlparser.Select)
	if !ok {
		return nil, errors.New("not a select statement")
	}

	var sc SelectClause

	// SELECT * ...
	// SELECT UserID,UpdatedAt ...
	for _, se := range s.SelectExprs {
		switch v := se.(type) {
		case *sqlparser.StarExpr:
			sc.SelectedFields = []string{}
		case *sqlparser.AliasedExpr:
			switch v := v.Expr.(type) {
			case *sqlparser.ColName:
				sc.SelectedFields = append(sc.SelectedFields, v.Name.String())
			case *sqlparser.FuncExpr:
				sc.SelectedFields = []string{}
				sc.Count = v.Name.String() == "count"
			}
		default:
			return nil, errors.New("unsupported select expression")
		}
	}

	// FROM users
	sc.Tablename = sqlparser.GetTableName(s.From[0].(*sqlparser.AliasedTableExpr).Expr).String()

	// WHERE
	sc.Matcher = q.And()
	if s.Where != nil {
		sc.Matcher = parsWhereExpr(s.Where.Expr)
	}

	// LIMIT 5
	// LIMIT 2,5
	if s.Limit != nil {
		if s.Limit.Offset != nil {
			sc.Skip = parseSQLVal(s.Limit.Offset.(*sqlparser.SQLVal)).(int)
		}
		sc.Limit = parseSQLVal(s.Limit.Rowcount.(*sqlparser.SQLVal)).(int)
	}

	// ORDER BY UpdatedAt
	// ORDER BY UpdatedAt DESC
	// ORDER BY UpdatedAt DESC, CreatedAt ASC     => All will be DESC due to strom limitation
	for _, ob := range s.OrderBy {
		if ob.Direction == "desc" {
			sc.OrderByReversed = true
		}
		sc.OrderBy = append(sc.OrderBy, ob.Expr.(*sqlparser.ColName).Name.String())
	}

	return &sc, nil
}

// FIXME replace panic by returned errors
func parsWhereExpr(expr sqlparser.Expr) q.Matcher {
	switch v := expr.(type) {
	//
	//
	//
	case *sqlparser.ComparisonExpr:
		field := v.Left.(*sqlparser.ColName).Name.String()
		var value any

		// Parse value
		switch sqlvalue := v.Right.(type) {
		case sqlparser.BoolVal:
			value = sqlvalue
		case sqlparser.ValTuple:
			var tuple []any
			for _, t := range sqlvalue {
				tuple = append(tuple, parseSQLVal(t.(*sqlparser.SQLVal)))
			}
			value = tuple
		case *sqlparser.SQLVal:
			value = parseSQLVal(sqlvalue)
		default:
			fmt.Printf("%#v\n", v)
			panic("unsupported Val")
		}

		// Parse operator
		switch v.Operator {
		case "=":
			return q.Eq(field, value)
		case "!=":
			return q.Not(q.Eq(field, value))
		case ">":
			return q.Gt(field, value)
		case ">=":
			return q.Gte(field, value)
		case "in":
			return q.In(field, value)
		case "<":
			return q.Lt(field, value)
		case "<=":
			return q.Lte(field, value)
		case "like":
			return q.Re(field, fmt.Sprintf("%v", value))
		default:
			fmt.Printf("%#v\n", v.Operator)
			panic("unsupported Operator")
		}
		//
		//
		//
	case *sqlparser.IsExpr:
		switch v.Operator {
		case "is not null":
			return q.Not(q.Eq(v.Expr.(*sqlparser.ColName).Name.String(), nil))
		default:
			fmt.Printf("%#v\n", v)
			panic("unsupported IsExpr")
		}
		//
		//
		//
	case *sqlparser.AndExpr:
		return q.And(
			parsWhereExpr(v.Left),
			parsWhereExpr(v.Right),
		)
		//
		//
		//
	case *sqlparser.OrExpr:
		return q.Or(
			parsWhereExpr(v.Left),
			parsWhereExpr(v.Right),
		)
		//
		//
		//
	default:
		fmt.Printf("%#v\n", v)
		panic("unsupported where expr type")
	}
}

func parseSQLVal(v *sqlparser.SQLVal) (value any) {
	switch v.Type {
	case sqlparser.StrVal:
		value = string(v.Val)

		// Try to convert to time.Time if possible
		if t, err := dateparse.ParseAny(string(v.Val)); err == nil {
			value = t.UTC()
		}
	case sqlparser.IntVal:
		value, _ = strconv.Atoi(string(v.Val))
	case sqlparser.FloatVal:
		value, _ = strconv.ParseFloat(string(v.Val), 64)
	case sqlparser.HexNum:
		value, _ = strconv.ParseInt(string(v.Val), 16, 64)
	case sqlparser.HexVal:
		b, err := v.HexDecode()
		if err != nil {
			panic(err)
		}
		value = b
	case sqlparser.ValArg:
		panic("unsupported ValArg") // TODO
	case sqlparser.BitVal:
		value = v.Val[0] == 1
	}

	return value
}
