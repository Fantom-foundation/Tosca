// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"fmt"
	"math"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
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

type revisionBounds struct{ min, max tosca.Revision }

func RevisionBounds(min, max tosca.Revision) Condition {
	if min > max {
		// This is an specification error, and should not be silently ignored
		panic(fmt.Sprintf("Invalid revision bounds: %v > %v", min, max))
	}
	return &revisionBounds{min, max}
}

func IsRevision(revision tosca.Revision) Condition {
	return RevisionBounds(revision, revision)
}

func IsRevisionCondition(condition Condition) bool {
	_, ok := condition.(*revisionBounds)
	return ok
}

// AnyKnownRevision restricts the revision to any revision covered by the CT specification.
func AnyKnownRevision() Condition {
	return RevisionBounds(MinRevision, NewestSupportedRevision)
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
	restrict := func(generator *gen.StateGenerator, revision tosca.Revision) {
		generator.SetRevision(revision)
	}
	res := []TestValue{}
	// If the revision is set to a specific value, only test this value,
	// except if it is the unknown next revision.
	if e.min == e.max && e.min != R99_UnknownNextRevision {
		res = append(res, NewTestValue(property, domain, e.min, restrict))
		return res
	}

	res = append(res, NewTestValue(property, domain, R99_UnknownNextRevision, restrict))
	for r := tosca.Revision(0); r <= NewestSupportedRevision; r++ {
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
	config   tosca.StorageStatus
	key      BindableExpression[U256]
	newValue BindableExpression[U256]
}

func StorageConfiguration(config tosca.StorageStatus, key, newValue BindableExpression[U256]) Condition {
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
	return gen.CheckStorageStatusConfig(c.config, s.Storage.GetOriginal(key), s.Storage.GetCurrent(key), newValue), nil
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
// Bind Transient Storage to non zero value

type bindTransientStorageToNonZero struct {
	key BindableExpression[U256]
}

func BindTransientStorageToNonZero(key BindableExpression[U256]) Condition {
	return &bindTransientStorageToNonZero{key}
}

func (c *bindTransientStorageToNonZero) Check(s *st.State) (bool, error) {
	key, err := c.key.Eval(s)
	if err != nil {
		return false, err
	}
	return !s.TransientStorage.IsZero(key), nil
}

func (c *bindTransientStorageToNonZero) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindTransientStorageToNonZero(key)
}

func (c *bindTransientStorageToNonZero) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, isNonZero bool) {
		key := c.key.GetVariable()
		c.key.BindTo(generator)
		if isNonZero {
			generator.BindTransientStorageToNonZero(key)
		} else {
			generator.BindTransientStorageToZero(key)
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *bindTransientStorageToNonZero) String() string {
	return fmt.Sprintf("Transient storage at [%v] is bound to non zero", c.key)
}

////////////////////////////////////////////////////////////
// Bind Transient Storage to zero value

type bindTransientStorageToZero struct {
	key BindableExpression[U256]
}

func BindTransientStorageToZero(key BindableExpression[U256]) Condition {
	return &bindTransientStorageToZero{key}
}

func (c *bindTransientStorageToZero) Check(s *st.State) (bool, error) {
	checked, err := (&bindTransientStorageToNonZero{c.key}).Check(s)
	return !checked, err
}

func (c *bindTransientStorageToZero) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindTransientStorageToZero(key)
}

func (c *bindTransientStorageToZero) GetTestValues() []TestValue {
	return BindTransientStorageToNonZero(c.key).GetTestValues()
}

func (c *bindTransientStorageToZero) String() string {
	return fmt.Sprintf("Transient storage at [%v] is bound to zero", c.key)
}

////////////////////////////////////////////////////////////
// Account Empty

type accountIsEmpty struct {
	address BindableExpression[U256]
}

func AccountIsEmpty(address BindableExpression[U256]) *accountIsEmpty {
	return &accountIsEmpty{address}
}

func (c *accountIsEmpty) Check(s *st.State) (bool, error) {
	address, err := c.address.Eval(s)
	if err != nil {
		return false, err
	}
	return s.Accounts.IsEmpty(NewAddress(address)), nil
}

func (c *accountIsEmpty) Restrict(generator *gen.StateGenerator) {
	address := c.address.GetVariable()
	c.address.BindTo(generator)
	generator.BindToAddressOfEmptyAccount(address)
}

func (c *accountIsEmpty) GetTestValues() []TestValue {
	property := Property(fmt.Sprintf("empty(%v)", c.address))
	restrict := func(generator *gen.StateGenerator, shouldBeEmpty bool) {
		if shouldBeEmpty {
			AccountIsEmpty(c.address).Restrict(generator)
		} else {
			AccountIsNotEmpty(c.address).Restrict(generator)
		}
	}
	return []TestValue{
		NewTestValue(property, boolDomain{}, true, restrict),
		NewTestValue(property, boolDomain{}, false, restrict),
	}
}

func (c *accountIsEmpty) String() string {
	return fmt.Sprintf("account_empty(%v)", c.address)
}

////////////////////////////////////////////////////////////
// Address not empty

type accountIsNotEmpty struct {
	address BindableExpression[U256]
}

func AccountIsNotEmpty(address BindableExpression[U256]) *accountIsNotEmpty {
	return &accountIsNotEmpty{address}
}

func (c *accountIsNotEmpty) Check(s *st.State) (bool, error) {
	res, err := AccountIsEmpty(c.address).Check(s)
	return !res, err
}

func (c *accountIsNotEmpty) Restrict(generator *gen.StateGenerator) {
	address := c.address.GetVariable()
	c.address.BindTo(generator)
	generator.BindToAddressOfNonEmptyAccount(address)
}

func (c *accountIsNotEmpty) GetTestValues() []TestValue {
	return AccountIsEmpty(c.address).GetTestValues()
}

func (c *accountIsNotEmpty) String() string {
	return fmt.Sprintf("!account_empty(%v)", c.address)
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
	property := Property(fmt.Sprintf("warm(%v)", c.key))
	return []TestValue{
		NewTestValue(property, boolDomain{}, true, restrictAccountWarmCold(c.key)),
	}
}

func (c *isAddressWarm) String() string {
	return fmt.Sprintf("account warm(%v)", c.key)
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
	res, err := IsAddressWarm(c.key).Check(s)
	return !res, err
}

func (c *isAddressCold) Restrict(generator *gen.StateGenerator) {
	key := c.key.GetVariable()
	c.key.BindTo(generator)
	generator.BindToColdAddress(key)
}

func (c *isAddressCold) GetTestValues() []TestValue {
	property := Property(fmt.Sprintf("warm(%v)", c.key))
	return []TestValue{
		NewTestValue(property, boolDomain{}, false, restrictAccountWarmCold(c.key)),
	}
}

func (c *isAddressCold) String() string {
	return fmt.Sprintf("account_cold(%v)", c.key)
}

func restrictAccountWarmCold(bindKey BindableExpression[U256]) func(generator *gen.StateGenerator, isWarm bool) {
	return func(generator *gen.StateGenerator, isWarm bool) {
		key := bindKey.GetVariable()
		bindKey.BindTo(generator)
		if isWarm {
			generator.BindToWarmAddress(key)
		} else {
			generator.BindToColdAddress(key)
		}
	}
}

////////////////////////////////////////////////////////////
// Has Self-Destructed

type hasSelfDestructed struct {
}

func HasSelfDestructed() Condition {
	return &hasSelfDestructed{}
}

func (c *hasSelfDestructed) Check(s *st.State) (bool, error) {
	return s.HasSelfDestructed, nil
}

func (c *hasSelfDestructed) Restrict(generator *gen.StateGenerator) {
	generator.MustBeSelfDestructed()
}

func (c *hasSelfDestructed) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, hasSelfDestructed bool) {
		if hasSelfDestructed {
			generator.MustBeSelfDestructed()
		} else {
			generator.MustNotBeSelfDestructed()
		}
	}
	return []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
}

func (c *hasSelfDestructed) String() string {
	return "hasSelfDestructed()"
}

////////////////////////////////////////////////////////////
// Has Not Self-Destructed

type hasNotSelfDestructed struct {
}

func HasNotSelfDestructed() Condition {
	return &hasNotSelfDestructed{}
}

func (c *hasNotSelfDestructed) Check(s *st.State) (bool, error) {
	return !s.HasSelfDestructed, nil
}

func (c *hasNotSelfDestructed) Restrict(generator *gen.StateGenerator) {
	generator.MustNotBeSelfDestructed()
}

func (c *hasNotSelfDestructed) GetTestValues() []TestValue {
	return HasSelfDestructed().GetTestValues()
}

func (c *hasNotSelfDestructed) String() string {
	return "hasNotSelfDestructed()"
}

////////////////////////////////////////////////////////////
// In Range 256 From Current Block

type inRange256FromCurrentBlock struct {
	blockNumber BindableExpression[U256]
}

func InRange256FromCurrentBlock(blockNumber BindableExpression[U256]) Condition {
	return &inRange256FromCurrentBlock{blockNumber}
}

func (c *inRange256FromCurrentBlock) Check(s *st.State) (bool, error) {
	paramBlockNumber, err := c.blockNumber.Eval(s)
	if err != nil {
		return false, err
	}
	if !paramBlockNumber.IsUint64() {
		return false, nil
	}
	uintParam := paramBlockNumber.Uint64()
	bottom := uint64(0)
	if s.BlockContext.BlockNumber > 256 {
		bottom = s.BlockContext.BlockNumber - 256
	}

	return bottom <= uintParam && uintParam < s.BlockContext.BlockNumber, nil
}

func (c *inRange256FromCurrentBlock) Restrict(generator *gen.StateGenerator) {
	paramVariable := c.blockNumber.GetVariable()
	c.blockNumber.BindTo(generator)
	generator.RestrictVariableToOneOfTheLast256Blocks(paramVariable)
}

func (c *inRange256FromCurrentBlock) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := BlockNumberOffsetDomain{}
	restrict := func(generator *gen.StateGenerator, offset int64) {
		paramVariable := c.blockNumber.GetVariable()
		c.blockNumber.BindTo(generator)
		generator.SetBlockNumberOffsetValue(paramVariable, offset)
	}
	testValues := []TestValue{}
	for _, value := range domain.SamplesForAll([]int64{}) {
		testValues = append(testValues, NewTestValue(property, domain, value, restrict))
	}
	return testValues
}

func (c *inRange256FromCurrentBlock) String() string {
	return c.blockNumber.String()
}

////////////////////////////////////////////////////////////
// Out Of Range 256 From Current Block

type outOfRange256FromCurrentBlock struct {
	blockNumber BindableExpression[U256]
}

func OutOfRange256FromCurrentBlock(blockNumber BindableExpression[U256]) Condition {
	return &outOfRange256FromCurrentBlock{blockNumber}
}

func (c *outOfRange256FromCurrentBlock) Check(s *st.State) (bool, error) {
	res, err := InRange256FromCurrentBlock(c.blockNumber).Check(s)
	return !res, err
}

func (c *outOfRange256FromCurrentBlock) Restrict(generator *gen.StateGenerator) {
	paramVariable := c.blockNumber.GetVariable()
	c.blockNumber.BindTo(generator)
	generator.RestrictVariableToNoneOfTheLast256Blocks(paramVariable)
}

func (c *outOfRange256FromCurrentBlock) GetTestValues() []TestValue {
	return InRange256FromCurrentBlock(c.blockNumber).GetTestValues()
}

func (c *outOfRange256FromCurrentBlock) String() string {
	return c.blockNumber.String()
}

////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////
// index Has a blob hash

type hasBlobHash struct {
	index BindableExpression[U256]
}

func HasBlobHash(blockNumber BindableExpression[U256]) Condition {
	return &hasBlobHash{blockNumber}
}

func (c *hasBlobHash) Check(s *st.State) (bool, error) {
	paramBlockNumber, err := c.index.Eval(s)
	if err != nil {
		return false, err
	}
	if !paramBlockNumber.IsUint64() {
		return false, nil
	}
	uintParam := paramBlockNumber.Uint64()
	return uintParam < uint64(len(s.TransactionContext.BlobHashes)), nil
}

func (c *hasBlobHash) Restrict(generator *gen.StateGenerator) {
	paramVariable := c.index.GetVariable()
	c.index.BindTo(generator)
	generator.IsPresentBlobHashIndex(paramVariable)
}

func (c *hasBlobHash) GetTestValues() []TestValue {
	property := Property(c.String())
	domain := boolDomain{}
	restrict := func(generator *gen.StateGenerator, hasBlobHash bool) {
		paramVariable := c.index.GetVariable()
		c.index.BindTo(generator)
		if hasBlobHash {
			generator.IsPresentBlobHashIndex(paramVariable)
		} else {
			generator.IsAbsentBlobHashIndex(paramVariable)
		}
	}
	testValues := []TestValue{
		NewTestValue(property, domain, true, restrict),
		NewTestValue(property, domain, false, restrict),
	}
	return testValues
}

func (c *hasBlobHash) String() string {
	return fmt.Sprintf("%v has BlobHash", c.index.String())
}

////////////////////////////////////////////////////////////
// index does not have a blob hash

type hasNoBlobHash struct {
	index BindableExpression[U256]
}

func HasNoBlobHash(index BindableExpression[U256]) Condition {
	return &hasNoBlobHash{index}
}

func (c *hasNoBlobHash) Check(s *st.State) (bool, error) {
	res, err := HasBlobHash(c.index).Check(s)
	return !res, err
}

func (c *hasNoBlobHash) Restrict(generator *gen.StateGenerator) {
	paramVariable := c.index.GetVariable()
	c.index.BindTo(generator)
	generator.IsAbsentBlobHashIndex(paramVariable)
}

func (c *hasNoBlobHash) GetTestValues() []TestValue {
	return HasBlobHash(c.index).GetTestValues()
}

func (c *hasNoBlobHash) String() string {
	return fmt.Sprintf("%v does not have BlobHash", c.index.String())
}

////////////////////////////////////////////////////////////
