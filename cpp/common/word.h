#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <initializer_list>
#include <ostream>

namespace tosca {

// A Word is a single 32-byte fixed-size byte vector without numerical interpretation. Words are trivial objects of
// fixed size that can be serialized by copying their byte pattern.
//
// By default, new instances are not initialized. Thus, when defining a local variable
//
//   Word word;  // the value is undefined
//
// the value is not defined. If zero-initialization is desired, an empty initializer list can be added:
//
//   Word word{};  // the value is zero
//
// This allows cases where an initialization is not required to skip it.
class Word {
 public:
  Word() = default;

  // A convenience constructor supporting integer literals for initializing word. Elements not listed in the initializer
  // will be initialized with zero. Elements exceeding the 32 entry limit will be ignored.
  Word(std::initializer_list<const std::uint8_t> list);

  // Supports equality comparison. Note: since words do not exhibit a numerical interpretation, the less-than order
  // would not be well defined.
  bool operator==(const Word&) const = default;

  // Prints a word as a hax string in upper case.
  friend std::ostream& operator<<(std::ostream&, const Word&);

 private:
  std::array<std::byte, 32> data_;
};

}  // namespace tosca
