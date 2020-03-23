module Pages.Users exposing (Model, Msg, page)

import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode as Encode
import Spa.Page
import Spa.Types as Types
import Url.Builder as Url
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Users Model Msg model msg appMsg
page =
    Spa.Page.element
        { title = always "Users"
        , init = init
        , update = update
        , subscriptions = always subscriptions
        , view = always view
        }



-- INIT


type alias Model =
    { users : List User
    , userEdits : Dict String User
    , error : Maybe Http.Error
    }


emptyUser =
    { id = ""
    , admin = False
    , email = ""
    , first = ""
    , last = ""
    }

init : Types.PageContext route Global.Model -> Params.Users -> ( Model, Cmd Msg )
init context _ =
    ( { users = []
      , userEdits = Dict.empty
      , error = Nothing
      }
    , case context.global of
        Global.SignedIn sess ->
            getUsers sess.authToken

        Global.SignedOut _ ->
            Cmd.none
    )



-- UPDATE


type Msg
    = UpdateUsers (Result Http.Error (List User))
    | PostUser String User
    | UserPosted String (Result Http.Error Response)
    | EditUser String User
    | DiscardUserEdits String
    | NewUser


type alias User =
    { id : String
    , first : String
    , last : String
    , email : String
    , admin : Bool

    --, roles : List String
    }


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg )
update context msg model =
    case msg of
        UpdateUsers (Ok users) ->
            ( { model | users = users }
            , Cmd.none
            )

        UpdateUsers (Err err) ->
            ( { model | error = Just err }
            , Cmd.none
            )

        UserPosted id (Ok _) ->
            ( { model | userEdits = Dict.remove id model.userEdits }
            , case context.global of
                Global.SignedIn sess ->
                    getUsers sess.authToken

                Global.SignedOut _ ->
                    Cmd.none
            )

        EditUser id user ->
            ( { model | userEdits = Dict.insert id user model.userEdits }
            , Cmd.none
            )

        DiscardUserEdits id ->
            ( { model | userEdits = Dict.remove id model.userEdits }
            , Cmd.none
            )

        PostUser id user ->
            ( model
            , case context.global of
                Global.SignedIn sess ->
                    postUser sess.authToken user.id user

                Global.SignedOut _ ->
                    Cmd.none
            )

        NewUser ->
            ( { model | users = emptyUser :: model.users }
            , Cmd.none
            )

        _ ->
            ( model
            , Cmd.none
            )


getUsers : String -> Cmd Msg
getUsers token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users" ] []
        , expect = Http.expectJson UpdateUsers usersDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


usersDecoder : Decode.Decoder (List User)
usersDecoder =
    Decode.list userDecoder


userDecoder =
    Decode.succeed User
        |> required "id" Decode.string
        |> required "firstName" Decode.string
        |> required "lastName" Decode.string
        |> required "email" Decode.string
        |> optional "admin" Decode.bool False



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> Element Msg
view model =
    column
        [ width fill, spacing 32 ]
        [ el [ padding 16, Font.size 24 ] <| text "Users"
        , viewError model.error
        , el [ width fill, Font.bold ] <| button "new user" palette.green NewUser
        , viewUsers model.userEdits model.users
        ]


viewError error =
    case error of
        Just (Http.BadUrl str) ->
            text <| "bad URL: " ++ str

        Just Http.Timeout ->
            text "timeout"

        Just Http.NetworkError ->
            text "network error"

        Just (Http.BadStatus status) ->
            text <| "bad status: " ++ String.fromInt status

        Just (Http.BadBody str) ->
            text <| "bad body: " ++ str

        Nothing ->
            none


viewUsers edits users =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map
            (\user -> viewUser
                (modified edits user)
                (userValue edits user))
            users


viewUser modded user =
    wrappedRow
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        buttonRow
                            [ button "discard" palette.pale <| DiscardUserEdits user.id
                            , button "save" palette.green <| PostUser user.id user
                            ]
                    ]

                else
                    []
               )
        )
        [ viewTextProperty
            { name = "First name"
            , value = user.first
            , action = \x -> EditUser user.id { user | first = x }
            }
        , viewTextProperty
            { name = "Last name"
            , value = user.last
            , action = \x -> EditUser user.id { user | last = x }
            }
        , viewTextProperty
            { name = "Email"
            , value = user.email
            , action = \x -> EditUser user.id { user | email = x }
            }
        , viewRoles
            [ { role = "user"
              , value = True
              , action = \x -> EditUser user.id user
              }
            , { role = "admin"
              , value = user.admin
              , action = \x -> EditUser user.id { user | admin = x }
              }
            ]
        ]


buttonRow : List (Element Msg) -> Element Msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button : String -> Color -> Msg -> Element Msg
button lbl color action =
    Input.button
        [ Background.color color
        , padding 16
        , width fill
        , Border.rounded 32
        ]
        { onPress = Just action
        , label = el [ centerX ] <| text lbl
        }


modified edits user =
    case Dict.get user.id edits of
        Just u ->
            u /= user

        Nothing ->
            False


userValue edits user =
    case Dict.get user.id edits of
        Just u ->
            u

        Nothing ->
            user


field edits user fld =
    case Dict.get user.id edits of
        Just u ->
            fld u

        Nothing ->
            fld user


viewTextProperty { name, value, action } =
    Input.text
        [ padding 16
        , width (fill |> minimum 150)
        , Border.width 0
        , Border.rounded 0
        , focused [ Background.color palette.yellow ]
        , Background.color palette.pale
        , spacing 0
        ]
        { onChange = action
        , text = value
        , placeholder = Nothing
        , label = label Input.labelAbove name
        }


label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color palette.gray
        ]
        << text


viewRoles =
    row
        []
        << List.map viewRole


viewRole { role, value, action } =
    Input.checkbox
        [ padding 16 ]
        { checked = value
        , icon = Input.defaultCheckbox
        , label = label Input.labelRight role
        , onChange = action
        }


postUser : String -> String -> User -> Cmd Msg
postUser token id user =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users", id ] []
        , expect = Http.expectJson (UserPosted id) responseDecoder
        , body = user |> userEncoder |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


userEncoder : User -> Encode.Value
userEncoder user =
    Encode.object
        [ ( "firstName", Encode.string user.first )
        , ( "lastName", Encode.string user.last )
        , ( "email", Encode.string user.email )
        ]


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""
