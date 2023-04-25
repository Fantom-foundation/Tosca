#include "common/word.h"

#include <memory>

#include "intx/intx.hpp"

namespace tosca {

static_assert(std::is_trivial_v<Word>);
static_assert(std::is_trivially_assignable_v<Word, Word>);
static_assert(sizeof(Word) == 32);

#ifndef __APPLE__  // std::endian not available on OSX
static_assert(std::endian::native == std::endian::little, "Only supporting little-endian systems so far");
#endif

// Cannot use std::bit_cast on OSX as of 2023-04-24, using reinterpret_cast
// instead.
static Word FromUint256(intx::uint256 v) { return reinterpret_cast<Word&>(v); }
static intx::uint256 ToUint256(Word v) { return reinterpret_cast<intx::uint256&>(v); }
static_assert(sizeof(intx::uint256) == sizeof(Word));

Word::Word(std::initializer_list<const std::uint8_t> list) {
  auto dst = data_.begin();
  auto src = std::rbegin(list);
  for (; dst != data_.end() && src != std::rend(list); ++dst, ++src) {
    *dst = static_cast<std::byte>(*src);
  }
  std::fill(dst, data_.end(), std::byte{0});
}

std::byte Word::operator[](std::uint8_t offset) const {
  if (offset < data_.size()) {
    return data_[(data_.size() - 1u) - offset];
  } else {
    return std::byte{0};
  }
}

std::byte Word::operator[](const Word& offset) const {
  const auto size = static_cast<std::uint8_t>(data_.size());

  if (offset < Word{size}) {
    return data_[(size - 1u) - static_cast<std::uint8_t>(offset.data_[0])];
  } else {
    return std::byte{0};
  }
}

std::strong_ordering Word::operator<=>(const Word& other) const {
  for (std::size_t i = data_.size() - 1; i > 0; --i) {
    if (data_[i] != other.data_[i]) {
      return data_[i] <=> other.data_[i];
    }
  }
  return data_[0] <=> other.data_[0];
}

Word Word::operator+(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a + b);
}

Word Word::operator-(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a - b);
}

Word Word::operator*(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a * b);
}

Word Word::operator/(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = ToUint256(*this);
  auto b = ToUint256(denom);
  return FromUint256(a / b);
}

Word Word::operator%(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = ToUint256(*this);
  auto b = ToUint256(denom);
  return FromUint256(a % b);
}

Word Word::operator<<(const Word& shift) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(shift);
  return FromUint256(a << b);
}

Word Word::operator>>(const Word& shift) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(shift);
  return FromUint256(a >> b);
}

Word Word::operator|(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a | b);
}

Word Word::operator&(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a & b);
}

Word Word::operator^(const Word& other) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(other);
  return FromUint256(a ^ b);
}

Word Word::operator~() const {
  auto a = ToUint256(*this);
  return FromUint256(~a);
}

Word Word::SignedDiv(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = ToUint256(*this);
  auto b = ToUint256(denom);
  return FromUint256(intx::sdivrem(a, b).quot);
}

Word Word::SignedMod(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = ToUint256(*this);
  auto b = ToUint256(denom);
  return FromUint256(intx::sdivrem(a, b).rem);
}

Word Word::Exp(const Word& exponent) const {
  auto a = ToUint256(*this);
  auto b = ToUint256(exponent);
  return FromUint256(intx::exp(a, b));
}

std::ostream& operator<<(std::ostream& out, const Word& word) {
  constexpr auto toSymbol = [](char x) -> char { return x < 10 ? '0' + x : 'A' + x - 10; };

  for (auto it = word.data_.rbegin(); it != word.data_.rend(); ++it) {
    out << toSymbol(static_cast<char>(*it >> 4));
    out << toSymbol(static_cast<char>(*it) & 0xF);
  }
  return out;
}

}  // namespace tosca
