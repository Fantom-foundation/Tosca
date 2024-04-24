//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package rlz

import (
	"fmt"
	"math"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Condition represents a state property.
type Condition interface {
	// Check evaluates the Condition.
	Check(*st.State) (bool, error)

	// Restrict sets constraints on the given StateGenerator such that this
	// Condition holds.
	Restrict(*gen.StateGenerator)

	// GetTestValues produces a list of test values probing the boundary defined
	// by this constraint.
	GetTestValues() []TestValue

	fmt.Stringer
}

////////////////////////////////////////////////////////////
// Conjunction

type conjunction struct {
	conditions []Condition
}

func And(conditions ...Condition) Condition {
	if len(conditions) == 1 {
		return conditions[0]
	}
	// Merge nested conjunctions into a single conjunction.
	res := []Condition{}
	for _, cur := range conditions {
		if c, ok := cur.(*conjunction); ok {
			res = append(res, c.conditions...)
		} else {
			res = append(res, cur)
		}
	}
	return &conjunction{conditions: res}
}

func (c *conjunction) Check(s *st.State) (bool, error) {
	for _, cur := range c.conditions {
		r, err := cur.Check(s)
		if !r || err != nil {
			return false, err
		}
	}
	return true, nil
}

func (c *conjunction) Restrict(generator *gen.StateGenerator) {
	for _, cur := range c.conditions {
		cur.Restrict(generator)
	}
}

func (c *conjunction) GetTestValues() []TestValue {
	res := []TestValue{}
	for _, cur := range c.conditions {
		res = append(res, cur.GetTestValues()...)
	}
	return res
}

func (c *conjunction) String() string {
	if len(c.conditions) == 0 {
		return "true"
	}
	first := true
	var builder strings.Builder
	for _, cur := range c.conditions {
		if !first {
			builder.WriteString(" ∧ ")
		} else {
			first = false
		}
		builder.WriteString(cur.String())
	}
	return builder.String()
}

////////////////////////////////////////////////////////////
// Equal

type eq[T any] struct {
	lhs Expression[T]
	rhs T
}

func Eq[T any](lhs Expression[T], rhs T) Condition {
	return &eq[T]{lhs, rhs}
}

func (e *eq[T]) Check(s *st.State) (bool, error) {
	domain := e.lhs.Domain()
	lhs, err := e.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Equal(lhs, e.rhs), nil
}

func (e *eq[T]) Restrict(generator *gen.StateGenerator) {
	e.lhs.Restrict(RestrictEqual, e.rhs, generator)
}

func (e *eq[T]) GetTestValues() []TestValue {
	property := e.lhs.Property()
	domain := e.lhs.Domain()
	restrict := func(generator *gen.StateGenerator, value T) {
		e.lhs.Restrict(RestrictEqual, value, generator)
	}
	res := []TestValue{}
	for _, value := range domain.Samples(e.rhs) {
		res = append(res, NewTestValue(property, domain, value, restrict))
	}
	return res
}

func (e *eq[T]) String() string {
	return fmt.Sprintf("%s = %v", e.lhs, e.rhs)
}

////////////////////////////////////////////////////////////
// Not Equal

type ne[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ne[T any](lhs Expression[T], rhs T) Condition {
	return &ne[T]{lhs, rhs}
}

func (e *ne[T]) Check(s *st.State) (bool, error) {
	domain := e.lhs.Domain()
	lhs, err := e.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !domain.Equal(lhs, e.rhs), nil
}

func (e *ne[T]) Restrict(generator *gen.StateGenerator) {
	domain := e.lhs.Domain()
	e.lhs.Restrict(RestrictEqual, domain.SomethingNotEqual(e.rhs), generator)
}

func (e *ne[T]) GetTestValues() []TestValue {
	return Eq(e.lhs, e.rhs).GetTestValues()
}

func (e *ne[T]) String() string {
	return fmt.Sprintf("%s ≠ %v", e.lhs, e.rhs)
}

////////////////////////////////////////////////////////////
// Less Than

type lt[T any] struct {
	lhs Expression[T]
	rhs T
}

func Lt[T any](lhs Expression[T], rhs T) Condition {
	return &lt[T]{lhs, rhs}
}

func (c *lt[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Less(lhs, c.rhs), nil
}

func (c *lt[T]) Restrict(generator *gen.StateGenerator) {
	domain := c.lhs.Domain()
	c.lhs.Restrict(RestrictLess, domain.Predecessor(c.rhs), generator)
}

func (c *lt[T]) GetTestValues() []TestValue {
	return Eq(c.lhs, c.rhs).GetTestValues()
}

func (c *lt[T]) String() string {
	return fmt.Sprintf("%s < %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Less Equal

type le[T any] struct {
	lhs Expression[T]
	rhs T
}

func Le[T any](lhs Expression[T], rhs T) Condition {
	return &le[T]{lhs, rhs}
}

func (c *le[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return domain.Less(lhs, c.rhs) || domain.Equal(lhs, c.rhs), nil
}

func (c *le[T]) Restrict(generator *gen.StateGenerator) {
	c.lhs.Restrict(RestrictLessEqual, c.rhs, generator)
}

func (c *le[T]) GetTestValues() []TestValue {
	return Eq(c.lhs, c.rhs).GetTestValues()
}

func (c *le[T]) String() string {
	return fmt.Sprintf("%s ≤ %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Greater Than

type gt[T any] struct {
	lhs Expression[T]
	rhs T
}

func Gt[T any](lhs Expression[T], rhs T) Condition {
	return &gt[T]{lhs, rhs}
}

func (c *gt[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !(domain.Less(lhs, c.rhs) || domain.Equal(lhs, c.rhs)), nil
}

func (c *gt[T]) Restrict(generator *gen.StateGenerator) {
	domain := c.lhs.Domain()
	c.lhs.Restrict(RestrictGreater, domain.Successor(c.rhs), generator)
}

func (c *gt[T]) GetTestValues() []TestValue {
	return Eq(c.lhs, c.rhs).GetTestValues()
}

func (c *gt[T]) String() string {
	return fmt.Sprintf("%s > %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Greater Equal

type ge[T any] struct {
	lhs Expression[T]
	rhs T
}

func Ge[T any](lhs Expression[T], rhs T) Condition {
	return &ge[T]{lhs, rhs}
}

func (c *ge[T]) Check(s *st.State) (bool, error) {
	domain := c.lhs.Domain()
	lhs, err := c.lhs.Eval(s)
	if err != nil {
		return false, err
	}
	return !domain.Less(lhs, c.rhs), nil
}

func (c *ge[T]) Restrict(generator *gen.StateGenerator) {
	c.lhs.Restrict(RestrictGreaterEqual, c.rhs, generator)
}

func (c *ge[T]) GetTestValues() []TestValue {
	return Eq(c.lhs, c.rhs).GetTestValues()
}

func (c *ge[T]) String() string {
	return fmt.Sprintf("%s ≥ %v", c.lhs, c.rhs)
}

////////////////////////////////////////////////////////////
// Revision Bounds

type revisionBounds struct{ min, max Revision }

func RevisionBounds(min, max Revision) Condition {
	if min > max {
		min, max = max, min
	}
	return &revisionBounds{min, max}
}

func IsRevision(revision Revision) Condition {
	return RevisionBounds(revision, revision)
}

func AnyKnownRevision() Condition {
	return RevisionBounds(Revision(0), R99_UnknownNextRevision-1)
}

func (c *revisionBounds) Check(s *st.State) (bool, error) {
	return c.min <= s.Revision && s.Revision <= c.max, nil
}

func (c *revisionBounds) Restrict(generator *gen.StateGenerator) {
	generator.AddRevisionBounds(c.min, c.max)
}

func (e *revisionBounds) GetTestValues() []TestValue {
	property := Property("revision")
	domain := revisionDomain{}
	restrict := func(generator *gen.StateGenerator, revision Revision) {
		generator.SetRevision(revision)
	}
	res := []TestValue{}
	for r := Revision(0); r <= R99_UnknownNextRevision; r++ {
		res = append(res, NewTestValue(property, domain, r, restrict))
	}
	return res
}

func (c *revisionBounds) String() string {
	if c.min == c.max {
		return fmt.Sprintf("revision(%v)", c.min)
	}
	return fmt.Sprintf("revision(%v-%v)", c.min, c.max)
}

////////////////////////////////////////////////////////////
// Is Code

type isCode struct {
	position BindableExpression[U256]
}

func IsCode(position BindableExpression[U256]) Condition {
	return &isCode{position}
}

func (c *isCode) Check(s *st.State) (bool, error) {
	pos, err := c.position.Eval(s)
	if err != nil {
		return false, err
	}
	if !pos.IsUint64() || pos.Uint64() > math.MaxInt {
		return true, nil // Out-of-bounds is considered code.
	}
	return s.Code.IsCode(int(pos.Uint64())), nil
}

func (c *isCode) Restrict(generator *gen.StateGenerator) {
	variable := c.position.GetVariable()
	c.position.BindTo(generator)
	generator.AddIsCode(variable)
}

func (c *isCode) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, isCode bool) {
		variable := c.position.GetVariable()
		c.position.BindTo(generator)
		if isCode {
			generator.AddIsCode(variable)
		} else {
			generator.AddIsData(variable)
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *isCode) String() string {
	return fmt.Sprintf("isCode[%s]", c.position)
}

////////////////////////////////////////////////////////////
// Is Data

type isData struct {
	position BindableExpression[U256]
}

func IsData(position BindableExpression[U256]) Condition {
	return &isData{position}
}

func (c *isData) Check(s *st.State) (bool, error) {
	pos, err := c.position.Eval(s)
	if err != nil {
		return false, err
	}
	if !pos.IsUint64() || pos.Uint64() > math.MaxInt {
		return false, nil // Out-of-bounds is considered code.
	}
	return s.Code.IsData(int(pos.Uint64())), nil
}

func (c *isData) Restrict(generator *gen.StateGenerator) {
	variable := c.position.GetVariable()
	c.position.BindTo(generator)
	generator.AddIsData(variable)
}

func (c *isData) GetTestValues() []TestValue {
	return IsCode(c.position).GetTestValues()
}

func (c *isData) String() string {
	return fmt.Sprintf("isData[%s]", c.position)
}

////////////////////////////////////////////////////////////
// Is Storage Warm

type isStorageWarm struct {
	key BindableExpression[U256]
}

func IsStorageWarm(key BindableExpression[U256]) Condition {
	return &isStorageWarm{key}
}

func (c *isStorageWarm) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	return s.Storage.IsWarm(key), nil
}

func (c *isStorageWarm) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindIsStorageWarm(key)
}

func (c *isStorageWarm) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, isWarm bool) {
		key := c.key.GetVariable()
		c.key.BindTo(generator)
		if isWarm {
			generator.BindIsStorageWarm(key)
		} else {
			generator.BindIsStorageCold(key)
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *isStorageWarm) String() string {
	return fmt.Sprintf("warm(%v)", c.key)
}

////////////////////////////////////////////////////////////
// Is Storage Cold

type isStorageCold struct {
	key BindableExpression[U256]
}

func IsStorageCold(key BindableExpression[U256]) Condition {
	return &isStorageCold{key}
}

func (c *isStorageCold) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	return !s.Storage.IsWarm(key), nil
}

func (c *isStorageCold) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindIsStorageCold(key)
}

func (c *isStorageCold) GetTestValues() []TestValue {
	return IsStorageWarm(c.key).GetTestValues()
}

func (c *isStorageCold) String() string {
	return fmt.Sprintf("cold(%v)", c.key)
}

////////////////////////////////////////////////////////////
// Storage Configuration

type storageConfiguration struct {
	config   gen.StorageCfg
	key      BindableExpression[U256]
	newValue BindableExpression[U256]
}

func StorageConfiguration(config gen.StorageCfg, key, newValue BindableExpression[U256]) Condition {
	return &storageConfiguration{config, key, newValue}
}

func (c *storageConfiguration) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	newValue, err := c.newValue.Eval(s)
	if err != nil {
		return false, err
	}
	return c.config.Check(s.Storage.GetOriginal(key), s.Storage.GetCurrent(key), newValue), nil
}

func (c *storageConfiguration) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)

	newValue := c.newValue.GetVariable()
	c.newValue.BindTo(generator)

	generator.BindStorageConfiguration(c.config, key, newValue)
}

func (c *storageConfiguration) GetTestValues() []TestValue {
	// For now, we only create the positive test case. It is assumed that all
	// storage configurations are covered by the specification.
	property := Property(c.String())
	domain := boolDomain{} // the domain is ignored
	restrict := func(generator *gen.StateGenerator, _ bool) {
		c.Restrict(generator)
	}
	return []TestValue{NewTestValue(property, domain, true, restrict)}
}

func (c *storageConfiguration) String() string {
	return fmt.Sprintf("StorageConfiguration(%v,%v,%v)", c.config, c.key, c.newValue)
}

////////////////////////////////////////////////////////////
// Is Address Warm

type isAddressWarm struct {
	key BindableExpression[U256]
}

func IsAddressWarm(key BindableExpression[U256]) Condition {
	return &isAddressWarm{key}
}

func (c *isAddressWarm) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	return s.Accounts.IsWarm(NewAddress(key)), nil
}

func (c *isAddressWarm) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindToWarmAddress(key)
}

func (c *isAddressWarm) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, isWarm bool) {
		key := c.key.GetVariable()
		c.key.BindTo(generator)
		if isWarm {
			generator.BindToWarmAddress(key)
		} else {
			generator.BindToColdAddress(key)
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *isAddressWarm) String() string {
	return fmt.Sprintf("warm(%v)", c.key)
}

////////////////////////////////////////////////////////////
// Is Address Cold

type isAddressCold struct {
	key BindableExpression[U256]
}

func IsAddressCold(key BindableExpression[U256]) Condition {
	return &isAddressCold{key}
}

func (c *isAddressCold) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	return s.Accounts.IsCold(NewAddress(key)), nil
}

func (c *isAddressCold) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindToColdAddress(key)
}

func (c *isAddressCold) GetTestValues() []TestValue {
	return IsAddressWarm(c.key).GetTestValues()
}

func (c *isAddressCold) String() string {
	return fmt.Sprintf("cold(%v)", c.key)
}

////////////////////////////////////////////////////////////
// Has Address Selfdestructed

type hasSelfDestructed struct {
	isSet bool
}

func HasSelfDestructed() Condition {
	return &hasSelfDestructed{true}
}

func (c *hasSelfDestructed) Check(s *st.State) (bool, error) {
	if c.isSet {
		return s.HasSelfDestructed, nil
	}
	return true, nil

}

func (c *hasSelfDestructed) Restrict(generator *gen.StateGenerator) {
	generator.SelfDestruct()
}

func (c *hasSelfDestructed) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, hasSelfDestructed bool) {
		if hasSelfDestructed {
			generator.hasSelfDestructedGen.MarkAsSelfDestructed()
		} else {
			generator.hasSelfDestructedGen.MarkAsNotSelfDestructed()
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *hasSelfDestructed) String() string {
	return fmt.Sprintf("hasSelfDestructed(%v)", c.isSet)
}

////////////////////////////////////////////////////////////
// Has Not Address Selfdestructed

type hasNotSelfDestructed struct {
	isSet bool
}

func HasNotSelfDestructed() Condition {
	return &hasNotSelfDestructed{true}
}

func (c *hasNotSelfDestructed) Check(s *st.State) (bool, error) {
	if c.isSet {
		return !s.HasSelfDestructed, nil
	}
	return true, nil
}

func (c *hasNotSelfDestructed) Restrict(generator *gen.StateGenerator) {
	generator.NotSelfDestruct()
}

func (c *hasNotSelfDestructed) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, hasSelfDestructed bool) {
		if hasSelfDestructed {
			generator.hasSelfDestructedGen.MarkAsSelfDestructed()
		} else {
			generator.hasSelfDestructedGen.MarkAsNotSelfDestructed()
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *hasNotSelfDestructed) String() string {
	return fmt.Sprintf("hasNotSelfDestructed(%v)", c.isSet)
}
