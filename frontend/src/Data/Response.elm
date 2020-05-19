module Data.Response exposing (Response)


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }
