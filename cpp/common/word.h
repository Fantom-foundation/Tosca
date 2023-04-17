#pragma once

#include <array>
#include <compare>
#include <cstddef>
#include <cstdint>
#include <initializer_list>
#include <ostream>

namespace tosca {

// A Word is a single 32-byte fixed-size byte vector representing an unsigned
// 256 bit integer. Words are trivial objects of fixed size that can be
// serialized by copying their byte pattern.
//
// By default, new instances are not initialized. Thus, when defining a local
// variable
//
//   Word word;  // the value is undefined
//
// the value is not defined. If zero-initialization is desired, an empty
// initializer list can be added:
//
//   Word word{};  // the value is zero
//
// This allows cases where an initialization is not required to skip it.
class Word {
 public:
  Word() = default;

  // A convenience constructor supporting integer literals for initializing
  // word. Elements not listed in the initializer will be initialized with zero.
  // The given sequence ends with the least significant byte. Elements beyond
  // the most significant byte (32) are ignored.
  Word(std::initializer_list<const std::uint8_t> list);

  // Byte offset starting from the most significant byte. Returns 0 on
  // out-of-bounds access.
  std::byte operator[](std::uint8_t) const;
  std::byte operator[](const Word&) const;

  bool operator==(const Word&) const = default;
  std::strong_ordering operator<=>(const Word& other) const;

  Word operator+(const Word&) const;
  Word operator-(const Word&) const;
  Word operator*(const Word&) const;

  // If the denominator is 0, the result will be 0.
  Word operator/(const Word&) const;
  Word operator%(const Word&) const;
  Word SignedDiv(const Word&) const;
  Word SignedMod(const Word&) const;

  Word operator<<(const Word&) const;
  Word operator>>(const Word&) const;
  Word operator|(const Word&) const;
  Word operator&(const Word&) const;
  Word operator^(const Word&) const;
  Word operator~() const;

  Word Exp(const Word&) const;

  // Prints a word as a hex string in upper case.
  friend std::ostream& operator<<(std::ostream&, const Word&);

  static const Word kMax;

 private:
  std::array<std::byte, 32> data_;
};

inline const Word Word::kMax = ~Word{0};

}  // namespace tosca
